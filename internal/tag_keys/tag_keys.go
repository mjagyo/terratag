package tag_keys

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

func GetTerratagAddedKey(filname string, resource string) string {
	return "terratag_added_" + filname + "_" + resource
}

func GetResourceExistingTagsKey(filename string, resource *hclwrite.Block) string {
	delimiter := "__"

	return "terratag_found_" + filename + delimiter + strings.Join(resource.Labels(), delimiter)
}
