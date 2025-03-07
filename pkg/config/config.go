package config

import (
	"os"

	v1 "github.com/jd-opensource/joylive-injector/client-go/apis/injector/v1"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	"go.uber.org/zap"
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
	NamespaceEnvName      = "JOYLIVE_NAMESPACE"
	MatchLabelsEnvName    = "JOYLIVE_MATCH_ENV_LABELS"
	DefaultNamespace      = "joylive"
	AgentVersionLabel     = "x-live-version"
)

var (
	Cert        string
	Key         string
	Addr        string
	MatchLabels string
)

// injection_deploy config
var (
	InitContainerName string
	// DefaultInjectorConfigMap store default injector configMap data
	DefaultInjectorConfigMap map[string]string
	// DefaultInjectorConfig define the default agent version information that can be injected into the application
	DefaultInjectorConfig *AgentInjectorConfig
	// InjectorConfigMaps key is configMap name, value is configMap data
	InjectorConfigMaps = map[string]map[string]string{}
	// InjectorAgentVersion store all versions of the agent and associated configuration information
	InjectorAgentVersion = map[string]v1.AgentVersionSpec{}
)

func init() {
	// Start the ConfigMap listener and initialize the content
	cmWatcher := NewConfigMapWatcher(resource.GetResource().ClientSet)
	err := cmWatcher.Start()
	if err != nil {
		log.Fatal("start cmWatcher error", zap.Error(err))
	}
	err = cmWatcher.InitConfigMap(GetNamespace())
	if err != nil {
		log.Fatal("init cm error", zap.Error(err))
	}

	// Start the AgentVersion listener and initialize the content
	avWatcher := NewAgentVersionWatcher(resource.GetResource().RestConfig)
	err = avWatcher.Start()
	if err != nil {
		log.Fatal("start avWatcher error", zap.Error(err))
	}
	err = avWatcher.InitAgentVersion(GetNamespace())
	if err != nil {
		log.Fatal("init agentVersion error", zap.Error(err))
	}

	// Initialize the default matchLabels from environment variables
	MatchLabels = os.Getenv(MatchLabelsEnvName)
}

func GetNamespace() string {
	namespace := os.Getenv(NamespaceEnvName)
	if len(namespace) == 0 {
		namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			log.Warnf("Failed to read namespace file: %v", err)
		}
		namespace = string(namespaceBytes)
	}
	if len(namespace) == 0 {
		return DefaultNamespace
	}
	return namespace
}
