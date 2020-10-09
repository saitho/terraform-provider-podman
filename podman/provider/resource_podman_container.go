package provider

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/podman/v2/pkg/specgen"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	spec "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/saitho/terraform-provider-podman/podman/client"
)

func resourcePodmanContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourcePodmanContainerCreate,
		Read:   resourcePodmanContainerRead,
		Update: resourcePodmanContainerUpdate,
		Delete: resourcePodmanContainerDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// ForceNew is not true for image because we need to
			// sane this against Docker image IDs, as each image
			// can have multiple names/tags attached do it.
			"image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// DiffSuppressFunc: suppressIfSHAwasAdded(), // TODO mvogel
			},

			"working_dir": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"capabilities": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				// TODO implement DiffSuppressFunc
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"add": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"drop": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"labels": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     labelSchema,
			},

			"entrypoint": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"command": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"env": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"mounts": {
				Type:        schema.TypeSet,
				Description: "Specification for mounts to be added to containers created as part of the service",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": {
							Type:        schema.TypeString,
							Description: "Container path",
							Required:    true,
						},
						"source": {
							Type:        schema.TypeString,
							Description: "Mount source (e.g. a volume name, a host path)",
							Optional:    true,
						},
						"type": {
							Type:             schema.TypeString,
							Description:      "The mount type",
							Required:         true,
							ValidateDiagFunc: validateStringMatchesPattern(`^(bind|volume|tmpfs)$`),
						},

						"rm": {
							Type:     schema.TypeBool,
							Default:  false,
							Optional: true,
						},

						"read_only": {
							Type:        schema.TypeBool,
							Description: "Whether the mount should be read-only",
							Optional:    true,
						},

						"bind_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the bind type",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"propagation": {
										Type:             schema.TypeString,
										Description:      "A propagation mode with the value",
										Optional:         true,
										ValidateDiagFunc: validateStringMatchesPattern(`^(private|rprivate|shared|rshared|slave|rslave)$`),
									},
								},
							},
						},
						"volume_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the volume type",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"no_copy": {
										Type:        schema.TypeBool,
										Description: "Populate volume with data from the target",
										Optional:    true,
									},
									"labels": {
										Type:        schema.TypeSet,
										Description: "User-defined key/value metadata",
										Optional:    true,
										Elem:        labelSchema,
									},
									"driver_name": {
										Type:        schema.TypeString,
										Description: "Name of the driver to use to create the volume.",
										Optional:    true,
									},
									"driver_options": {
										Type:        schema.TypeMap,
										Description: "key/value map of driver specific options",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"tmpfs_options": {
							Type:        schema.TypeList,
							Description: "Optional configuration for the tmpfs type",
							Optional:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"size_bytes": {
										Type:        schema.TypeInt,
										Description: "The size for the tmpfs mount in bytes",
										Optional:    true,
									},
									"mode": {
										Type:        schema.TypeInt,
										Description: "The permission mode for the tmpfs mount in an integer",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
			"volumes": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_container": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"container_path": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"host_path": {
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateDockerContainerPath,
						},

						"volume_name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"read_only": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"healthcheck": {
				Type:        schema.TypeList,
				Description: "A test to perform to check that the container is healthy",
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"test": {
							Type:        schema.TypeList,
							Description: "The test to perform as list",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"interval": {
							Type:             schema.TypeString,
							Description:      "Time between running the check (ms|s|m|h)",
							Optional:         true,
							Default:          "0s",
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"timeout": {
							Type:             schema.TypeString,
							Description:      "Maximum time to allow one check to run (ms|s|m|h)",
							Optional:         true,
							Default:          "0s",
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"start_period": {
							Type:             schema.TypeString,
							Description:      "Start period for the container to initialize before counting retries towards unstable (ms|s|m|h)",
							Optional:         true,
							Default:          "0s",
							ValidateDiagFunc: validateDurationGeq0(),
						},
						"retries": {
							Type:             schema.TypeInt,
							Description:      "Consecutive failures needed to report unhealthy",
							Optional:         true,
							Default:          0,
							ValidateDiagFunc: validateIntegerGeqThan(0),
						},
					},
				},
			},

			"log_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"privileged": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"dns": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_opts": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_search": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"publish_all_ports": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"restart": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "no",
				ValidateDiagFunc: validateStringMatchesPattern(`^(no|on-failure|always|unless-stopped)$`),
			},

			"max_retry_count": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"ports": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},

						"external": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"ip": {
							Type:     schema.TypeString,
							Default:  "0.0.0.0",
							Optional: true,
							ForceNew: true,
							StateFunc: func(val interface{}) string {
								// Empty IP assignments default to 0.0.0.0
								if val.(string) == "" {
									return "0.0.0.0"
								}

								return val.(string)
							},
						},

						"protocol": {
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"shm_size": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				ValidateDiagFunc: validateIntegerGeqThan(0),
			},

			"network_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, oldV, newV string, d *schema.ResourceData) bool {
					// treat "" as "default", which is Docker's default value
					if oldV == "" {
						oldV = "default"
					}
					if newV == "" {
						newV = "default"
					}
					return oldV == newV
				},
			},

			"pid_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"userns_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"sysctls": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"ipc_mode": {
				Type:        schema.TypeString,
				Description: "IPC sharing mode for the container",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
			"group_add": {
				Type:        schema.TypeSet,
				Description: "Additional groups for the container user",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"log_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"networks_advanced": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"aliases": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"ipv4_address": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"ipv6_address": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func stringSetToStringMap(stringSlice *schema.Set, separator string) map[string]string {
	return stringSliceToStringMap(stringSetToStringSlice(stringSlice), separator)
}

func stringSliceToStringMap(stringSlice []string, separator string) map[string]string {
	ret := map[string]string{}
	for _, s := range stringSlice {
		slice := strings.Split(s, separator)
		ret[strings.TrimSpace(slice[0])] = strings.TrimSpace(slice[1])
	}
	return ret
}

func resourcePodmanContainerCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	podmanClient := client.Client{}
	if err := podmanClient.Connect(); err != nil {
		return err
	}
	image := d.Get("image").(string)
	err = podmanClient.PullImage(image)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to create container with image %s: %s", image, err))
	}

	config := specgen.NewSpecGenerator(image, false)

	config.Privileged = d.Get("privileged").(bool)
	config.PublishExposedPorts = d.Get("publish_all_ports").(bool)
	config.RestartPolicy = d.Get("restart").(string)
	config.RestartRetries = d.Get("max_retry_count").(*uint)
	config.Remove = d.Get("rm").(bool)
	config.ReadOnlyFilesystem = d.Get("read_only").(bool)
	config.LogConfiguration = &specgen.LogConfig{
		Driver: d.Get("log_driver").(string),
	}

	if v, ok := d.GetOk("env"); ok {
		config.Env = stringSetToStringMap(v.(*schema.Set), "=")
	}

	if v, ok := d.GetOk("command"); ok {
		config.Command = stringListToStringSlice(v.([]interface{}))
		for _, v := range config.Command {
			if v == "" {
				return fmt.Errorf("values for command may not be empty")
			}
		}
	}

	if v, ok := d.GetOk("entrypoint"); ok {
		config.Entrypoint = stringListToStringSlice(v.([]interface{}))
	}

	if v, ok := d.GetOk("user"); ok {
		config.User = v.(string)
	}

	if v, ok := d.GetOk("ports"); ok {
		config.PortMappings = portSetToPodmanPortMappings(v.([]interface{}))
	}
	if v, ok := d.GetOk("working_dir"); ok {
		config.WorkDir = v.(string)
	}
	if v, ok := d.GetOk("host"); ok {
		config.HostAdd = stringSetToStringSlice(v.(*schema.Set))
	}

	volumes := []*specgen.NamedVolume{}
	volumesFrom := []string{}

	if v, ok := d.GetOk("volumes"); ok {
		volumes, volumesFrom, err = volumeSetToPodmanVolumes(v.(*schema.Set))
		if err != nil {
			return fmt.Errorf("Unable to parse volumes: %s", err)
		}
	}
	if len(volumes) != 0 {
		config.Volumes = volumes
	}

	if v, ok := d.GetOk("labels"); ok {
		config.Labels = labelSetToMap(v.(*schema.Set))
	}

	if value, ok := d.GetOk("healthcheck"); ok {
		config.HealthConfig = &manifest.Schema2HealthConfig{}

		if len(value.([]interface{})) > 0 {
			for _, rawHealthCheck := range value.([]interface{}) {
				rawHealthCheck := rawHealthCheck.(map[string]interface{})
				if testCommand, ok := rawHealthCheck["test"]; ok {
					config.HealthConfig.Test = stringListToStringSlice(testCommand.([]interface{}))
				}
				if rawInterval, ok := rawHealthCheck["interval"]; ok {
					config.HealthConfig.Interval, _ = time.ParseDuration(rawInterval.(string))
				}
				if rawTimeout, ok := rawHealthCheck["timeout"]; ok {
					config.HealthConfig.Timeout, _ = time.ParseDuration(rawTimeout.(string))
				}
				if rawStartPeriod, ok := rawHealthCheck["start_period"]; ok {
					config.HealthConfig.StartPeriod, _ = time.ParseDuration(rawStartPeriod.(string))
				}
				if rawRetries, ok := rawHealthCheck["retries"]; ok {
					config.HealthConfig.Retries, _ = rawRetries.(int)
				}
			}
		}
	}

	var mounts []spec.Mount

	if value, ok := d.GetOk("mounts"); ok {
		for _, rawMount := range value.(*schema.Set).List() {
			rawMount := rawMount.(map[string]interface{})
			mountInstance := spec.Mount{
				Destination: rawMount["target"].(string),
				Type:        rawMount["type"].(string),
				Source:      rawMount["source"].(string),
			}
			if value, ok := rawMount["read_only"]; ok {
				optionValue := "rw"
				if value.(bool) {
					optionValue = "ro"
				}
				mountInstance.Options = append(mountInstance.Options, optionValue)
			}

			// if mountType == "bind" {
			// 	if value, ok := rawMount["bind_options"]; ok {
			// 		if len(value.([]interface{})) > 0 {
			// 			mountInstance.BindOptions = &mount.BindOptions{}
			// 			for _, rawBindOptions := range value.([]interface{}) {
			// 				rawBindOptions := rawBindOptions.(map[string]interface{})
			// 				if value, ok := rawBindOptions["propagation"]; ok {
			// 					mountInstance.BindOptions.Propagation = mount.Propagation(value.(string))
			// 				}
			// 			}
			// 		}
			// 	}
			// } else if mountType == "volume" {
			// 	if value, ok := rawMount["volume_options"]; ok {
			// 		if len(value.([]interface{})) > 0 {
			// 			mountInstance.VolumeOptions = &mount.VolumeOptions{}
			// 			for _, rawVolumeOptions := range value.([]interface{}) {
			// 				rawVolumeOptions := rawVolumeOptions.(map[string]interface{})
			// 				if value, ok := rawVolumeOptions["no_copy"]; ok {
			// 					mountInstance.VolumeOptions.NoCopy = value.(bool)
			// 				}
			// 				if value, ok := rawVolumeOptions["labels"]; ok {
			// 					mountInstance.VolumeOptions.Labels = labelSetToMap(value.(*schema.Set))
			// 				}
			// 				// because it is not possible to nest maps
			// 				if value, ok := rawVolumeOptions["driver_name"]; ok {
			// 					if mountInstance.VolumeOptions.DriverConfig == nil {
			// 						mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
			// 					}
			// 					mountInstance.VolumeOptions.DriverConfig.Name = value.(string)
			// 				}
			// 				if value, ok := rawVolumeOptions["driver_options"]; ok {
			// 					if mountInstance.VolumeOptions.DriverConfig == nil {
			// 						mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
			// 					}
			// 					mountInstance.VolumeOptions.DriverConfig.Options = mapTypeMapValsToString(value.(map[string]interface{}))
			// 				}
			// 			}
			// 		}
			// 	}
			// } else if mountType == "tmpfs" {
			// 	if value, ok := rawMount["tmpfs_options"]; ok {
			// 		if len(value.([]interface{})) > 0 {
			// 			mountInstance.TmpfsOptions = &mount.TmpfsOptions{}
			// 			for _, rawTmpfsOptions := range value.([]interface{}) {
			// 				rawTmpfsOptions := rawTmpfsOptions.(map[string]interface{})
			// 				if value, ok := rawTmpfsOptions["size_bytes"]; ok {
			// 					mountInstance.TmpfsOptions.SizeBytes = (int64)(value.(int))
			// 				}
			// 				if value, ok := rawTmpfsOptions["mode"]; ok {
			// 					mountInstance.TmpfsOptions.Mode = os.FileMode(value.(int))
			// 				}
			// 			}
			// 		}
			// 	}
			// }

			mounts = append(mounts, mountInstance)
		}
		config.Mounts = mounts
	}

	if len(volumesFrom) != 0 {
		config.VolumesFrom = volumesFrom
	}

	if v, ok := d.GetOk("capabilities"); ok {
		for _, capInt := range v.(*schema.Set).List() {
			capa := capInt.(map[string]interface{})
			config.CapAdd = stringSetToStringSlice(capa["add"].(*schema.Set))
			config.CapDrop = stringSetToStringSlice(capa["drop"].(*schema.Set))
			break
		}
	}

	if v, ok := d.GetOk("dns"); ok {
		config.DNSServers = stringSetToDNSServers(v.(*schema.Set))
	}

	if v, ok := d.GetOk("dns_opts"); ok {
		config.DNSOptions = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("dns_search"); ok {
		config.DNSSearch = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("shm_size"); ok {
		size := int64(v.(int)) * 1024 * 1024
		config.ShmSize = &size
	}

	if v, ok := d.GetOk("log_opts"); ok {
		config.LogConfiguration.Options = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("network_mode"); ok {
		config.NetNS = specgen.Namespace{
			NSMode: specgen.NamespaceMode(v.(string)),
		}
	}

	if v, ok := d.GetOk("userns_mode"); ok {
		config.UserNS = specgen.Namespace{
			NSMode: specgen.NamespaceMode(v.(string)),
		}
	}
	if v, ok := d.GetOk("pid_mode"); ok {
		config.PidNS = specgen.Namespace{
			NSMode: specgen.NamespaceMode(v.(string)),
		}
	}

	if v, ok := d.GetOk("sysctls"); ok {
		config.Sysctl = mapTypeMapValsToString(v.(map[string]interface{}))
	}
	if v, ok := d.GetOk("ipc_mode"); ok {
		config.IpcNS = specgen.Namespace{
			NSMode: specgen.NamespaceMode(v.(string)),
		}
	}
	if v, ok := d.GetOk("group_add"); ok {
		config.Groups = stringSetToStringSlice(v.(*schema.Set))
	}

	var containerId string

	config.Name = d.Get("name").(string)

	if containerId, err = podmanClient.CreateContainer(config); err != nil {
		return fmt.Errorf("Unable to create container: %s", err)
	}

	d.SetId(containerId)

	//	if v, ok := d.GetOk("networks_advanced"); ok {
	//		if err := client.NetworkDisconnect(context.Background(), "bridge", containerId, false); err != nil {
	//			if !strings.Contains(err.Error(), "is not connected to the network bridge") {
	//				return fmt.Errorf("Unable to disconnect the default network: %s", err)
	//			}
	//		}
	//
	//		for _, rawNetwork := range v.(*schema.Set).List() {
	//			networkID := rawNetwork.(map[string]interface{})["name"].(string)
	//
	//			endpointConfig := &network.EndpointSettings{}
	//			endpointIPAMConfig := &network.EndpointIPAMConfig{}
	//			if v, ok := rawNetwork.(map[string]interface{})["aliases"]; ok {
	//				endpointConfig.Aliases = stringSetToStringSlice(v.(*schema.Set))
	//			}
	//			if v, ok := rawNetwork.(map[string]interface{})["ipv4_address"]; ok {
	//				endpointIPAMConfig.IPv4Address = v.(string)
	//			}
	//			if v, ok := rawNetwork.(map[string]interface{})["ipv6_address"]; ok {
	//				endpointIPAMConfig.IPv6Address = v.(string)
	//			}
	//			endpointConfig.IPAMConfig = endpointIPAMConfig
	//
	//			if err := client.NetworkConnect(context.Background(), networkID, retContainer.ID, endpointConfig); err != nil {
	//				return fmt.Errorf("Unable to connect to network '%s': %s", networkID, err)
	//			}
	//		}
	//	}
	//
	//	if v, ok := d.GetOk("upload"); ok {
	//
	//		var mode int64
	//		for _, upload := range v.(*schema.Set).List() {
	//			content := upload.(map[string]interface{})["content"].(string)
	//			contentBase64 := upload.(map[string]interface{})["content_base64"].(string)
	//			source := upload.(map[string]interface{})["source"].(string)
	//
	//			testParams := []string{content, contentBase64, source}
	//			setParams := 0
	//			for _, v := range testParams {
	//				if v != "" {
	//					setParams++
	//				}
	//			}
	//
	//			if setParams == 0 {
	//				return fmt.Errorf("error with upload content: one of 'content', 'content_base64', or 'source' must be set")
	//			}
	//			if setParams > 1 {
	//				return fmt.Errorf("error with upload content: only one of 'content', 'content_base64', or 'source' can be set")
	//			}
	//
	//			var contentToUpload string
	//			if content != "" {
	//				contentToUpload = content
	//			}
	//			if contentBase64 != "" {
	//				decoded, _ := base64.StdEncoding.DecodeString(contentBase64)
	//				contentToUpload = string(decoded)
	//			}
	//			if source != "" {
	//				sourceContent, err := ioutil.ReadFile(source)
	//				if err != nil {
	//					return fmt.Errorf("could not read file: %s", err)
	//				}
	//				contentToUpload = string(sourceContent)
	//			}
	//			file := upload.(map[string]interface{})["file"].(string)
	//			executable := upload.(map[string]interface{})["executable"].(bool)
	//
	//			buf := new(bytes.Buffer)
	//			tw := tar.NewWriter(buf)
	//			if executable {
	//				mode = 0744
	//			} else {
	//				mode = 0644
	//			}
	//			hdr := &tar.Header{
	//				Name: file,
	//				Mode: mode,
	//				Size: int64(len(contentToUpload)),
	//			}
	//			if err := tw.WriteHeader(hdr); err != nil {
	//				return fmt.Errorf("Error creating tar archive: %s", err)
	//			}
	//			if _, err := tw.Write([]byte(contentToUpload)); err != nil {
	//				return fmt.Errorf("Error creating tar archive: %s", err)
	//			}
	//			if err := tw.Close(); err != nil {
	//				return fmt.Errorf("Error creating tar archive: %s", err)
	//			}
	//
	//			dstPath := "/"
	//			uploadContent := bytes.NewReader(buf.Bytes())
	//			options := types.CopyToContainerOptions{}
	//			if err := client.CopyToContainer(context.Background(), retContainer.ID, dstPath, uploadContent, options); err != nil {
	//				return fmt.Errorf("Unable to upload volume content: %s", err)
	//			}
	//		}
	//	}

	if d.Get("start").(bool) {
		if err := podmanClient.StartContainer(containerId); err != nil {
			return fmt.Errorf("Unable to start container: %s", err)
		}
	}

	if d.Get("attach").(bool) {
		var b bytes.Buffer

		//		ctx := context.Background()

		//		if d.Get("logs").(bool) {
		//			go func() {
		//				reader, err := client.ContainerLogs(ctx, retContainer.ID, types.ContainerLogsOptions{
		//					ShowStdout: true,
		//					ShowStderr: true,
		//					Follow:     true,
		//					Timestamps: false,
		//				})
		//				if err != nil {
		//					log.Panic(err)
		//				}
		//				defer reader.Close()
		//
		//				scanner := bufio.NewScanner(reader)
		//				for scanner.Scan() {
		//					line := scanner.Text()
		//					b.WriteString(line)
		//					b.WriteString("\n")
		//
		//					log.Printf("[DEBUG] container logs: %s", line)
		//				}
		//				if err := scanner.Err(); err != nil {
		//					log.Fatal(err)
		//				}
		//			}()
		//		}

		if err := podmanClient.WaitContainer(containerId); err != nil {
			return fmt.Errorf("Unable to wait container end of execution: %s", err)
		} else {
			if d.Get("logs").(bool) {
				d.Set("container_logs", b.String())
			}
		}
	}

	return resourcePodmanContainerRead(d, meta)
}

func resourcePodmanContainerRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourcePodmanContainerUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourcePodmanContainerDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
