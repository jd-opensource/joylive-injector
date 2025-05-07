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
	ClusterIdEnvName       = "JOYLIVE_CLUSTER_ID"
	DefaultNamespace       = "joylive"
	AgentVersionLabel      = "x-live-version"
	ServiceNameLabel       = "jmsf.jd.com/service"
	ServiceGroupLabel      = "jmsf.jd.com/group"
	ServiceSpaceLabel      = "jmsf.jd.com/space"
	JdapServiceSpaceLabel  = "jmsf.jd.com/service-space"
	ApplicationLabel       = "jmsf.jd.com/app"
	EnhanceTypeLabel       = "jmsf.jd.com/enhance"
	EnhanceTypeAgent       = "agent"
	EnhanceTypeSidecar     = "sidecar"
	SidecarEnhanceLabel    = "sidecar.istio.io/inject"
	ApmTypeLabel           = "jmsf.jd.com/apm"
	TenantLabel            = "jmsf.jd.com/tenant"
	JdapApplicationLabel   = "app.jdap.io/name"
	WebHookMatchKeyEnv     = "JOYLIVE_MATCH_KEY"
	WebHookMatchValueEnv   = "JOYLIVE_MATCH_VALUE"
)

var (
	Cert              string
	Key               string
	Addr              string
	MatchLabels       string
	ControlPlaneUrl   string
	ClusterId         string
	WebHookMatchKey   string
	WebHookMatchValue string
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
	// Initialize the default cluster id from environment variables
	ClusterId = os.Getenv(ClusterIdEnvName)
	WebHookMatchKey = os.Getenv(WebHookMatchKeyEnv)
	WebHookMatchValue = os.Getenv(WebHookMatchValueEnv)
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
