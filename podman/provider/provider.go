package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"registry_auth": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Address of the registry",
						},

						"username": {
							Type:        schema.TypeString,
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
							Description: "Username for the registry",
						},

						"password": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Description: "Password for the registry",
						},
					},
				},
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"podman_container": resourcePodmanContainer(),
			// "podman_image":     resourcePodmanImage(),
			// "podman_network":   resourcePodmanNetwork(),
			// "podman_volume":    resourcePodmanVolume(),
		},
	}
}
