package resource

import (
	"encoding/json"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"io"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"strings"

	"github.com/jd-opensource/joylive-injector/pkg/config"
)

const (
	SettingTypeLabel       = "label"
	SettingTypeEnvironment = "environment"
	SettingTypeAnnotation  = "annotation"
)

type Response struct {
	Error error       `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type ApplicationEnvResponse struct {
	Response
	Data map[string]string `json:"data"`
}

type SettingConfigResponse struct {
	Response
	Data []map[string]string `json:"data"`
}

func GetApplicationEnvironments(labels map[string]string) (map[string]string, error) {
	serviceSpace := labels[config.ServiceSpaceLabel]
	application := labels[config.ApplicationLabel]
	if len(serviceSpace) == 0 {
		serviceSpace = labels[config.JdapServiceSpaceLabel]
	}
	if len(application) == 0 {
		application = labels[config.JdapApplicationLabel]
	}
	envMaps := make(map[string]string)
	if len(serviceSpace) != 0 && len(application) != 0 {
		url := fmt.Sprintf(
			"%s/ns/%s/application/%s/environments", config.ControlPlaneUrl, serviceSpace, application,
		)
		resp, err := http.Get(url)
		if err != nil {
			return nil, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed, status: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var response ApplicationEnvResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, err
		}

		if response.Error != nil {
			return nil, response.Error
		}
		for key, value := range response.Data {
			if len(config.FilterSensitive) != 0 && config.FilterSensitive == "true" {
				// Filter out keys ending with "USERNAME" and "PASSWORD" to enhance security.
				if strings.HasSuffix(key, "USERNAME") || strings.HasSuffix(key, "PASSWORD") {
					continue
				}
			}
			envMaps[key] = value
		}
		envMaps["APPLICATION_NAME"] = application
		envMaps["APPLICATION_SERVICE_NAMESPACE"] = serviceSpace
	}

	if v, ok := labels[config.RegisterTypeLabel]; ok {
		envMaps["CONFIG_REGISTRY_ENABLED"] = v
	}

	if v, ok := labels[config.ConfigureTypeLabel]; ok {
		envMaps["CONFIG_CENTER_ENABLED"] = v
	}

	if group, ok := labels[config.ServiceGroupLabel]; ok {
		envMaps["APPLICATION_SERVICE_GROUP"] = group
	} else {
		envMaps["APPLICATION_SERVICE_GROUP"] = "default"
	}

	if service, ok := labels[config.ServiceNameLabel]; ok {
		envMaps["APPLICATION_SERVICE_NAME"] = service
	}

	//envMaps["APPLICATION_LOCATION_CLUSTER"] = config.ClusterId

	if swimlane, ok := labels[config.SwimLaneLabel]; ok {
		envMaps["CONFIG_LANE_ENABLED"] = "true"
		envMaps["APPLICATION_LOCATION_LANE"] = swimlane
	}

	return envMaps, nil
}

// GetSidecarConfig istio config info
func GetSidecarConfig(pod corev1.Pod, deploymentName string) (SidecarConfig, error) {
	sidecarConfig := SidecarConfig{
		Labels:      make(map[string]string),
		Envs:        make(map[string]string),
		Annotations: make(map[string]string),
	}
	// get service id
	serviceSpace, serviceId := extractServiceInfo(pod.Labels)

	// check
	if len(serviceSpace) == 0 || len(serviceId) == 0 || len(config.ClusterId) == 0 || len(deploymentName) == 0 {
		return sidecarConfig, nil
	}

	// get config
	settingData, err := getSettingData(serviceSpace, serviceId, config.ClusterId, deploymentName)
	if err != nil {
		return sidecarConfig, err
	}

	// handle config
	processSettingData(settingData, &sidecarConfig)

	return sidecarConfig, nil
}

func extractServiceInfo(podLabels map[string]string) (string, string) {
	if podLabels == nil {
		return "", ""
	}
	serviceSpace := podLabels[config.ServiceSpaceLabel]
	if serviceSpace == "" {
		serviceSpace = podLabels[config.JdapServiceSpaceLabel]
	}
	serviceId := podLabels[config.ServiceNameLabel]
	return serviceSpace, serviceId
}

func getSettingData(serviceSpace, serviceId, clusterId, deploymentName string) ([]map[string]string, error) {
	// check
	if len(serviceSpace) == 0 || len(serviceId) == 0 || len(clusterId) == 0 || len(deploymentName) == 0 {
		return nil, nil
	}

	// URL
	url := fmt.Sprintf("%s/spaces/%s/clusters/%s/services/%s/workloads/%s/setting",
		config.ControlPlaneUrl, serviceSpace, clusterId, serviceId, deploymentName)
	log.Debugf("[mutation] /injection-pod: sidecar setting data url: %s", url)

	resp, err := http.Get(url)
	if err != nil || resp == nil || resp.Body == nil {
		log.Warnf("Failed to get sidecar setting data:", err)
		return nil, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response SettingConfigResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("API error: %w", response.Error)
	}

	return response.Data, nil
}

func processSettingData(settingData []map[string]string, config *SidecarConfig) {
	if len(settingData) == 0 {
		return
	}
	for _, setting := range settingData {
		settingType, exists := setting["type"]
		if !exists {
			continue
		}

		key, keyExists := setting["key"]
		value, valueExists := setting["value"]
		if !keyExists || !valueExists {
			continue
		}

		switch settingType {
		case SettingTypeLabel:
			config.Labels[key] = value
		case SettingTypeEnvironment:
			config.Envs[key] = value
		case SettingTypeAnnotation:
			config.Annotations[key] = value
		}
	}
}

type SidecarConfig struct {
	Labels      map[string]string `json:"labels"`
	Envs        map[string]string `json:"envs"`
	Annotations map[string]string `json:"annotations"`
}
