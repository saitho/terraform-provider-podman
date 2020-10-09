package provider

import (
	"errors"
	"github.com/containers/podman/v2/pkg/specgen"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net"
)

func stringSetToStringSlice(stringSet *schema.Set) []string {
	var ret []string
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		ret = append(ret, envVal.(string))
	}
	return ret
}

func stringListToStringSlice(stringList []interface{}) []string {
	var ret []string
	for _, v := range stringList {
		if v == nil {
			ret = append(ret, "")
			continue
		}
		ret = append(ret, v.(string))
	}
	return ret
}

func portSetToPodmanPortMappings(ports []interface{}) []specgen.PortMapping {
	var retPortMappings []specgen.PortMapping

	for _, portInt := range ports {
		port := portInt.(map[string]interface{})
		portMapping := specgen.PortMapping{}
		portMapping.Protocol = port["protocol"].(string)
		internal, intOk := port["internal"].(uint16)
		if intOk {
			portMapping.ContainerPort = internal
		}
		external, extOk := port["external"].(uint16)
		if extOk {
			portMapping.HostPort = external
		}

		ip, ipOk := port["ip"].(string)
		if ipOk {
			portMapping.HostIP = ip
		}

		if intOk && (extOk || ipOk) {
			retPortMappings = append(retPortMappings, portMapping)
		}
	}

	return retPortMappings
}

func volumeSetToPodmanVolumes(volumes *schema.Set) ([]*specgen.NamedVolume, []string, error) {
	var retVolumes []*specgen.NamedVolume
	var retVolumeFromContainers []string

	for _, volumeInt := range volumes.List() {
		volume := volumeInt.(map[string]interface{})
		fromContainer := volume["from_container"].(string)
		containerPath := volume["container_path"].(string)
		volumeName := volume["volume_name"].(string)
		if len(volumeName) == 0 {
			volumeName = volume["host_path"].(string)
		}
		readOnly := volume["read_only"].(bool)

		switch {
		case len(fromContainer) == 0 && len(containerPath) == 0:
			return retVolumes, retVolumeFromContainers, errors.New("Volume entry without container path or source container")
		case len(fromContainer) != 0 && len(containerPath) != 0:
			return retVolumes, retVolumeFromContainers, errors.New("Both a container and a path specified in a volume entry")
		case len(fromContainer) != 0:
			retVolumeFromContainers = append(retVolumeFromContainers, fromContainer)
		case len(volumeName) != 0:
			readWrite := "rw"
			if readOnly {
				readWrite = "ro"
			}
			namedVolume := &specgen.NamedVolume{
				Name:    volumeName,
				Dest:    containerPath,
				Options: []string{readWrite},
			}
			retVolumes = append(retVolumes, namedVolume)
		}
	}

	return retVolumes, retVolumeFromContainers, nil
}

func mapTypeMapValsToString(typeMap map[string]interface{}) map[string]string {
	mapped := make(map[string]string, len(typeMap))
	for k, v := range typeMap {
		mapped[k] = v.(string)
	}
	return mapped
}

func stringSetToDNSServers(stringSet *schema.Set) []net.IP {
	var ret []net.IP
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		ret = append(ret, envVal.(net.IP))
	}
	return ret
}
