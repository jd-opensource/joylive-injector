package config

import (
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	"os"
)

const (
	// DefaultConfigPath is the default path to the configuration file
	DefaultConfigPath     = "/etc"
	AgentConfig           = "config.yaml"
	AgentInjectConfigName = "agent.yaml"
	LogConfig             = "logback.xml"
	BootConfig            = "bootstrap.properties"
	ConfigMountPath       = "/joylive/config"
	EmptyDirMountPath     = "/joylive"
	InitEmptyDirMountPath = "/agent"
	ConfigMapEnvName      = "JOYLIVE_CONFIGMAP_NAME"
)

var (
	Cert               string
	Key                string
	Addr               string
	ConfigPath         string
	ConfigMountSubPath string
	MatchLabel         string
)

// injection_deploy config
var (
	InitContainerName string
	//InitContainerImage    string
	//InitContainerCmd      string
	//InitContainerArgs     string
	//InitContainerEnvKey   string
	//InitContainerEnvValue string
	InjectorConfigMap map[string]string
	InjectorConfig    *AgentInjectConfig
)

func init() {
	cmWatcher := NewConfigMapWatcher(resource.GetResource().ClientSet)
	err := cmWatcher.Start()
	if err != nil {
		panic(err.Error())
	}
	err = cmWatcher.InitConfigMap(GetNamespace(), os.Getenv(ConfigMapEnvName))
	if err != nil {
		panic(err.Error())
	}
}

func GetNamespace() string {
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatalf("Failed to read namespace file: %v", err)
		panic(err.Error())
	}
	return string(namespace)
}
