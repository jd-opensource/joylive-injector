package config

import (
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AgentInjectConfig struct {
	Image    string `yaml:"image"`
	Version  string `yaml:"version"`
	Cmd      string `yaml:"cmd"`
	Args     string `yaml:"args"`
	EnvKey   string `yaml:"envKey"`
	EnvValue string `yaml:"envValue"`
}

func GetAgentInjectConfig(yamlData string) (*AgentInjectConfig, error) {
	var config AgentInjectConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		log.Error("Error parsing YAML", zap.Error(err))
		return nil, err
	}
	return &config, nil
}
