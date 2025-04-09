package config

import (
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AgentInjectorConfig struct {
	AgentConfig Agent `yaml:"agent"`
}

type Agent struct {
	Image   string            `yaml:"image"`
	Version string            `yaml:"version"`
	Envs    map[string]string `json:"envs"`
}

func GetAgentInjectConfig(yamlData string) (*AgentInjectorConfig, error) {
	var config AgentInjectorConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		log.Error("Error parsing YAML", zap.Error(err))
		return nil, err
	}
	return &config, nil
}
