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

func GetApplicationEnvironments(namespace, application string) (map[string]string, error) {
	url := fmt.Sprintf(
		"%s/v1/ns/%s/application/%s/environments",
		config.ControlPlaneUrl, namespace, application,
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
	return response.Data, nil
}
