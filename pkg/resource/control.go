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
	serviceSpace, application := labels[config.ServiceSpaceLabel], labels[config.ApplicationLabel]
	if len(serviceSpace) == 0 || len(application) == 0 {
		serviceSpace, application = labels["jmsf.jd.com/service-space"], labels["app.jdap.io/name"]
	}
	envMaps := make(map[string]string)
	if len(serviceSpace) != 0 && len(application) != 0 {
		url := fmt.Sprintf(
			"%s/ns/%s/application/%s/environments",
			config.ControlPlaneUrl, serviceSpace, application,
		)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
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
	envMaps["APPLICATION_LOCATION_CLUSTER"] = config.ClusterId

	return envMaps, nil
}
