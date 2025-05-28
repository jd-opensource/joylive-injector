package resource

import (
	"encoding/json"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"io"
	"net/http"
)

type Response struct {
	Error error       `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type ApplicationEnvResponse struct {
	Response
	Data map[string]string `json:"data"`
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
			envMaps[key] = value
		}
		envMaps["APPLICATION_NAME"] = application
		envMaps["APPLICATION_SERVICE_NAMESPACE"] = serviceSpace
	}

	if v, ok := labels[config.RegisterTypeLabel]; ok {
		envMaps["CONFIG_REGISTRY_ENABLED"] = v
	}

	if group, ok := labels[config.ServiceGroupLabel]; ok {
		envMaps["APPLICATION_SERVICE_GROUP"] = group
	} else {
		envMaps["APPLICATION_SERVICE_GROUP"] = "default"
	}

	if service, ok := labels[config.ServiceNameLabel]; ok {
		envMaps["APPLICATION_SERVICE_NAME"] = service
	}

	envMaps["APPLICATION_LOCATION_CLUSTER"] = config.ClusterId

	if swimlane, ok := labels[config.SwimLaneLabel]; ok {
		envMaps["CONFIG_LANE_ENABLED"] = "true"
		envMaps["APPLICATION_LOCATION_LANE"] = swimlane
	}

	return envMaps, nil
}
