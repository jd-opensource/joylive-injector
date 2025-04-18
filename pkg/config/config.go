package config

import (
	"os"

	v1 "github.com/jd-opensource/joylive-injector/client-go/apis/injector/v1"
	"github.com/jd-opensource/joylive-injector/pkg/log"
)

const (
	AgentConfig            = "config.yaml"
	InjectorConfigName     = "injector.yaml"
	LogConfig              = "logback.xml"
	BootConfig             = "bootstrap.properties"
	ConfigMountPath        = "/joylive/config"
	EmptyDirMountPath      = "/joylive"
	InitEmptyDirMountPath  = "/agent"
	InitContainerCmd       = "/bin/sh"
	InitContainerArgs      = "-c, cp -r /joylive/* /agent && chmod -R 777 /agent"
	ConfigMapEnvName       = "JOYLIVE_CONFIGMAP_NAME"
	NamespaceEnvName       = "JOYLIVE_NAMESPACE"
	MatchLabelsEnvName     = "JOYLIVE_MATCH_ENV_LABELS"
	ControlPlaneUrlEnvName = "JOYLIVE_CONTROL_PLANE_URL"
	DefaultNamespace       = "joylive"
	AgentVersionLabel      = "x-live-version"
	ServiceSpaceLabel      = "jmsf.jd.com/space"
	ApplicationLabel       = "jmsf.jd.com/application"
)

var (
	Cert            string
	Key             string
	Addr            string
	MatchLabels     string
	ControlPlaneUrl string
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
	// Initialize the default matchLabels from environment variables
	MatchLabels = os.Getenv(MatchLabelsEnvName)
	// Initialize the default control plane url from environment variables
	ControlPlaneUrl = os.Getenv(ControlPlaneUrlEnvName)
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
