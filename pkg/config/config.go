package config

import (
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"os"
	"path/filepath"
)

const (
	// DefaultConfigPath is the default path to the configuration file
	DefaultConfigPath = "/etc"
	AgentConfig       = "config.yaml"
	LogConfig         = "logback.xml"
	BootConfig        = "bootstrap.properties"
)

var (
	Cert               string
	Key                string
	Addr               string
	ConfigPath         string
	ConfigMountPath    string
	EmptyDirMountPath  string
	ConfigMountSubPath string
	MatchLabel         string
)

// injection_deploy config
var (
	InitContainerName     string
	InitContainerImage    string
	InitContainerCmd      string
	InitContainerArgs     string
	InitContainerEnvKey   string
	InitContainerEnvValue string
	InitEmptyDirMountPath string
)

func ReadConfigs() (map[string]string, error) {
	configMap := make(map[string]string)
	configFile, err := os.ReadFile(filepath.Join(ConfigPath, AgentConfig))
	if err != nil {
		log.Infof("Error reading logback.xml: %v", err)
		return nil, err
	}
	configMap[AgentConfig] = string(configFile)
	// Read logback.xml
	xmlFile, err := os.ReadFile(filepath.Join(ConfigPath, LogConfig))
	if err != nil {
		log.Infof("Error reading logback.xml: %v", err)
		return nil, err
	}
	configMap[LogConfig] = string(xmlFile)

	// Read bootstrap.properties
	propsFile, err := os.ReadFile(filepath.Join(ConfigPath, BootConfig))
	if err != nil {
		log.Infof("Error reading bootstrap.properties: %v", err)
		return nil, err
	}
	configMap[BootConfig] = string(propsFile)
	return configMap, nil
}
