package tagging

import (
	"log"
	"sync"

	"github.com/env0/terratag/internal/common"
	"github.com/env0/terratag/internal/convert"
	"github.com/env0/terratag/internal/tag_keys"
	"github.com/env0/terratag/internal/terraform"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var resourceType string

func defaultTaggingFn(args TagBlockArgs) (*Result, error) {
	tagBlock, err := TagBlock(args)
	if err != nil {
		return nil, err
	}

	return &Result{SwappedTagsStrings: []string{tagBlock}}, nil
}

func ParseHclValueStringToTokens(hclValueString string) hclwrite.Tokens {
	file, diags := hclwrite.ParseConfig([]byte("tempKey = "+hclValueString), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		log.Print("error parsing hcl value string " + hclValueString)
		panic(diags.Errs()[0])
	}

	tempAttribute := file.Body().GetAttribute("tempKey")

	return tempAttribute.Expr().BuildTokens(hclwrite.Tokens{})
}

func TagBlock(args TagBlockArgs) (string, error) {
	resourceLabels := args.Block.Labels()
	var resourceLabel string

	if len(resourceLabels) > 0 {
		resourceLabel = resourceLabels[0]
		resourceType = resourceLabels[0]
	} else {
		getResourceType := GetResourceTypeForBlock()
		resourceLabel = getResourceType(args.Block.Labels())
	}

	hasExistingTags, err := convert.MoveExistingTags(args.Filename, args.Terratag, args.Block, args.TagId)
	if err != nil {
		return "", err
	}

	terratagAddedKey := "local." + tag_keys.GetTerratagAddedKey(args.Filename, resourceLabel)
	newTagsValue := terratagAddedKey

	if hasExistingTags {
		existingTagsKey := tag_keys.GetResourceExistingTagsKey(args.Filename, args.Block)
		existingTagsExpression := convert.GetExistingTagsExpression(args.Terratag.Found[existingTagsKey])
		newTagsValue = "merge( " + existingTagsExpression + ", " + terratagAddedKey + ")"
	}

	newTagsValueTokens := ParseHclValueStringToTokens(newTagsValue)
	args.Block.Body().SetAttributeRaw(args.TagId, newTagsValueTokens)

	return newTagsValue, nil
}

func HasResourceTagFn(resourceType string) bool {
	return resourceTypeToFnMap[resourceType] != nil
}

func TagResource(args TagBlockArgs) (*Result, error) {
	resourceType := terraform.GetResourceType(*args.Block)

	customTaggingFn := resourceTypeToFnMap[resourceType]

	if customTaggingFn != nil {
		return customTaggingFn(args)
	} else {
		return defaultTaggingFn(args)
	}
}

func GetResourceTypeForBlock() func([]string) string {
	var mu sync.Mutex // Mutex to protect access to storedValue

	return func(arg []string) string {
		mu.Lock()         // Lock before modifying or accessing shared state
		defer mu.Unlock() // Unlock after the operation is complete

		return resourceType
	}
}

var resourceTypeToFnMap = map[string]TagResourceFn{
	"aws_autoscaling_group":      tagAutoscalingGroup,
	"aws_instance":               tagAwsInstance,
	"google_container_cluster":   tagContainerCluster,
	"azurerm_kubernetes_cluster": tagAksK8sCluster,
}

type TagBlockArgs struct {
	Filename string
	Block    *hclwrite.Block
	Tags     string
	Terratag common.TerratagLocal
	TagId    string
}

type TagResourceFn func(args TagBlockArgs) (*Result, error)

type Result struct {
	SwappedTagsStrings []string
}
