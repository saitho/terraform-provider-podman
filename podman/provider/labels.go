package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

var labelSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"label": &schema.Schema{
			Type:        schema.TypeString,
			Description: "Name of the label",
			Required:    true,
		},
		"value": &schema.Schema{
			Type:        schema.TypeString,
			Description: "Value of the label",
			Required:    true,
		},
	},
}

func labelToPair(label map[string]interface{}) (string, string) {
	return label["label"].(string), label["value"].(string)
}

func labelSetToMap(labels *schema.Set) map[string]string {
	labelsSlice := labels.List()

	mapped := make(map[string]string, len(labelsSlice))
	for _, label := range labelsSlice {
		l, v := labelToPair(label.(map[string]interface{}))
		mapped[l] = v
	}
	return mapped
}
