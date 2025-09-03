package config

import (
	"os"

	v1 "github.com/jd-opensource/joylive-injector/client-go/apis/injector/v1"
	"github.com/jd-opensource/joylive-injector/pkg/log"
)

const (
	AgentConfig             = "config.yaml"
	InjectorConfigName      = "injector.yaml"
	LogConfig               = "logback.xml"
	BootConfig              = "bootstrap.properties"
	ConfigMountPath         = "/joylive/config"
	EmptyDirMountPath       = "/joylive"
	InitEmptyDirMountPath   = "/agent"
	InitContainerCmd        = "/bin/sh"
	InitContainerArgs       = "-c, cp -r /joylive/* /agent && chmod -R 777 /agent"
	DefaultConfigMapEnvName = "JOYLIVE_CONFIGMAP_NAME"
	RuleConfigMapEnvName    = "JOYLIVE_RULE_CONFIGMAP_NAME"
	NamespaceEnvName        = "JOYLIVE_NAMESPACE"
	MatchLabelsEnvName      = "JOYLIVE_MATCH_ENV_LABELS"
	ControlPlaneUrlEnvName  = "JOYLIVE_CONTROL_PLANE_URL"
	FilterSensitiveEnvName  = "JOYLIVE_FILTER_SENSITIVE"
	ClusterIdEnvName        = "JOYLIVE_CLUSTER_ID"
	DefaultNamespace        = "joylive"
	AgentVersionLabel       = "x-live-version"
	ServiceNameLabel        = "jmsf.jd.com/service"
	ServiceGroupLabel       = "jmsf.jd.com/group"
	ServiceSpaceLabel       = "jmsf.jd.com/space"
	JdapServiceSpaceLabel   = "jmsf.jd.com/service-space"
	ApplicationLabel        = "jmsf.jd.com/app"
	EnhanceTypeLabel        = "jmsf.jd.com/enhance"
	RegisterTypeLabel       = "jmsf.jd.com/register"
	EnhanceTypeAgent        = "agent"
	EnhanceTypeSidecar      = "sidecar"
	SidecarEnhanceLabel     = "sidecar.istio.io/inject"
	ApmTypeLabel            = "jmsf.jd.com/apm"
	TenantLabel             = "jmsf.jd.com/tenant"
	SwimLaneLabel           = "jmsf.jd.com/swimlane"
	JdapApplicationLabel    = "app.jdap.io/name"
	WebHookMatchKeyEnv      = "JOYLIVE_MATCH_KEY"
	WebHookMatchValueEnv    = "JOYLIVE_MATCH_VALUE"
)

var (
	Cert              string
	Key               string
	Addr              string
	MatchLabels       string
	ControlPlaneUrl   string
	FilterSensitive   string
	ClusterId         string
	WebHookMatchKey   string
	WebHookMatchValue string
	ConfigMapName     = "joylive-injector-config"
	RuleConfigMapName = "joylive-injector-rules"
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
	InjectorRules        = map[string]*InjectorRule{}
)

func init() {
	// Initialize the default matchLabels from environment variables
	MatchLabels = os.Getenv(MatchLabelsEnvName)
	// Initialize the default control plane url from environment variables
	ControlPlaneUrl = os.Getenv(ControlPlaneUrlEnvName)
	// Initialize the switch for filter sensitive from environment variables
	FilterSensitive = os.Getenv(FilterSensitiveEnvName)
	// Initialize the default cluster id from environment variables
	ClusterId = os.Getenv(ClusterIdEnvName)
	WebHookMatchKey = os.Getenv(WebHookMatchKeyEnv)
	WebHookMatchValue = os.Getenv(WebHookMatchValueEnv)
	ConfigMapName = os.Getenv(DefaultConfigMapEnvName)
	RuleConfigMapName = os.Getenv(RuleConfigMapEnvName)
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
