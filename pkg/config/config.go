package config

import (
	v1 "github.com/jd-opensource/joylive-injector/client-go/apis/injector/v1"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	"os"
)

const (
	AgentConfig           = "config.yaml"
	InjectorConfigName    = "injector.yaml"
	LogConfig             = "logback.xml"
	BootConfig            = "bootstrap.properties"
	ConfigMountPath       = "/joylive/config"
	EmptyDirMountPath     = "/joylive"
	InitEmptyDirMountPath = "/agent"
	InitContainerCmd      = "/bin/sh"
	InitContainerArgs     = "-c, cp -r /joylive/* /agent && chmod -R 777 /agent"
	ConfigMapEnvName      = "JOYLIVE_CONFIGMAP_NAME"
	AgentVersionLabel     = "x-live-version"
	LiveSpaceIdLabel      = "x-live-space-id"
	LiveUnitLabel         = "x-live-unit"
	LiveCellLabel         = "x-live-cell"
)

var (
	Cert               string
	Key                string
	Addr               string
	ConfigMountSubPath string
	MatchLabel         string
)

// injection_deploy config
var (
	InitContainerName        string
	DefaultInjectorConfigMap map[string]string
	InjectorConfigMaps       map[string]map[string]string
	InjectorConfig           *AgentInjectorConfig
	InjectorAgentVersion     map[string]v1.AgentVersionSpec
)

func init() {
	// Start the ConfigMap listener and initialize the content
	cmWatcher := NewConfigMapWatcher(resource.GetResource().ClientSet)
	err := cmWatcher.Start()
	if err != nil {
		panic(err.Error())
	}
	err = cmWatcher.InitConfigMap(GetNamespace())
	if err != nil {
		panic(err.Error())
	}

	// Start the AgentVersion listener and initialize the content
	avWatcher := NewAgentVersionWatcher(resource.GetResource().RestConfig)
	err = avWatcher.Start()
	if err != nil {
		panic(err.Error())
	}
	err = avWatcher.InitAgentVersion(GetNamespace())
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
