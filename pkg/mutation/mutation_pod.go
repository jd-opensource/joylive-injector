package mutation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/jd-opensource/joylive-injector/pkg/admission"
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	jsoniter "github.com/json-iterator/go"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AgentVolumeName       = "joylive-agent-dir"
	MutatingWebhookConfig = "mutating-webhook-config"
	DefaultCPU            = "200m"
	DefaultMemory         = "300Mi"
)

func init() {
	admission.Register(admission.AdmissionFunc{
		Type: admission.AdmissionTypeMutating,
		Path: "/injection-pod",
		Func: injectionPod,
	})
}

// injectionPod Handling admission webhook requests, mainly used to handle Pod resources
func injectionPod(request *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	switch request.Kind.Kind {
	case "Pod":
		log.Debugf("[mutation] /injection-pod: received request: %v,the operition is %s ", request.Resource, request.Operation)
		if request.Operation == "DELETE" {
			log.Debugf("[mutation] /injection-pod: received delete request name is : %s, namespace is %s ", request.Name, request.Namespace)
			return &admissionv1.AdmissionResponse{
				Allowed: true,
				Result: &metav1.Status{
					Code:    http.StatusOK,
					Message: "success",
				},
			}, nil
		}
		var pod corev1.Pod
		err := jsoniter.Unmarshal(request.Object.Raw, &pod)
		if err != nil {
			errMsg := fmt.Sprintf("[mutation] /injection-pod: failed to unmarshal object: %v", err)
			log.Error(errMsg)
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: errMsg,
				},
			}, nil
		}

		envs := makePodEnvs(&pod)
		targetPod := pod.DeepCopy()

		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}

		if targetPod.Labels == nil {
			targetPod.Labels = make(map[string]string)
		}

		addEnvs, addLabels := matchRule(pod.Labels)
		if len(addEnvs) > 0 {
			log.Infof("[mutation] /injection-pod: add envs %v to pod %s/%s by injector rule", addEnvs, pod.Name, pod.Namespace)
			for k, v := range addEnvs {
				// Check if the environment variable already exists
				exists := false
				for _, env := range pod.Spec.Containers[0].Env {
					if env.Name == k {
						exists = true
						break
					}
				}
				// If it exists, skip adding it
				if exists {
					continue
				}
				// Add the environment variable
				targetPod.Spec.Containers[0].Env = append(targetPod.Spec.Containers[0].Env, corev1.EnvVar{Name: k, Value: v})
			}
		}

		if len(addLabels) > 0 {
			log.Infof("[mutation] /injection-pod: add labels %v to pod %s/%s by injector rule", addLabels, pod.Name, pod.Namespace)
			for k, v := range addLabels {
				// Check if the label already exists
				if _, exists := targetPod.Labels[k]; !exists {
					targetPod.Labels[k] = v
				}
			}
		}

		rs := resource.GetResource()
		deploymentName, err := rs.GetDeploymentName(&pod, request.Namespace)
		if err != nil {
			errMsg := fmt.Sprintf("[mutation] /injection-pod: failed to get deployment by pod: %v", err)
			log.Error(errMsg)
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: errMsg,
				},
			}, nil
		}

		targetPod.Spec.InitContainers = addPodInitContainer(targetPod, envs, deploymentName)
		targetPod.Spec.Containers = modifyPodContainer(targetPod, envs, deploymentName)
		targetPod.Spec.Volumes = addPodVolume(targetPod, deploymentName)

		log.Debug("[mutation] /injection-pod: add configmap volume finished")
		// path
		patchStr, err := createPodPatch(targetPod, &pod)
		if err != nil {
			return &admissionv1.AdmissionResponse{
				UID:     request.UID,
				Allowed: true,
			}, nil
		}
		return &admissionv1.AdmissionResponse{
			UID:     request.UID,
			Allowed: true,
			Patch:   patchStr,
			PatchType: func() *admissionv1.PatchType {
				pt := admissionv1.PatchTypeJSONPatch
				return &pt
			}(),
		}, nil
	default:
		return &admissionv1.AdmissionResponse{
			UID:     request.UID,
			Allowed: true,
		}, nil
	}
}

func makePodEnvs(pod *corev1.Pod) []corev1.EnvVar {
	rawPrefixes := config.MatchLabels
	// Split multiple prefix configurations
	prefixes := strings.Split(rawPrefixes, ",")
	validPrefixes := make([]string, 0, len(prefixes))
	// Clean up and validate prefixes
	for _, p := range prefixes {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			validPrefixes = append(validPrefixes, trimmed)
		}
	}
	if len(validPrefixes) == 0 {
		log.Warnf("[mutation] /injection-pod: no valid prefix configured in MatchLabels")
		return nil
	}
	var envs []corev1.EnvVar
	// Iterate through all labels, matching multiple prefixes
	for labelKey, labelValue := range pod.Labels {
		if labelValue == "" {
			continue
		}
		for _, prefix := range validPrefixes {
			if strings.HasPrefix(labelKey, prefix) {
				envs = append(envs, corev1.EnvVar{
					Name:  labelKey,
					Value: labelValue,
				})
				break
			}
		}
	}
	envs = append(envs,
		corev1.EnvVar{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		corev1.EnvVar{
			Name:  "NODE_ZONES",
			Value: getNodeWithZone(),
		})
	return envs
}

func getNodeWithZone() string {
	nodes, err := resource.GetResource().GetNodes()
	if err != nil {
		log.Error("[mutation] /injection-pod: failed to get nodes", zap.Error(err))
		return ""
	}

	// Get all zones from all nodes
	zoneMap := make(map[string][]string)
	for _, node := range nodes.Items {
		zone := node.Labels["topology.jdos.io/zone"]
		if zone != "" {
			zoneMap[zone] = append(zoneMap[zone], node.Name)
		}
	}

	var zoneList []string
	for zone, nodeNames := range zoneMap {
		zoneList = append(zoneList, zone+":"+strings.Join(nodeNames, ","))
	}

	return strings.Join(zoneList, ";")
}

func addPodInitContainer(targetPod *corev1.Pod, _ []corev1.EnvVar, deploymentName string) []corev1.Container {
	initContainers := targetPod.Spec.InitContainers
	for _, container := range initContainers {
		if container.Name == config.InitContainerName {
			log.Warnf("[mutation] /injection-pod: A container [%s] already exists, skipping the addition logic.", config.InitContainerName)
			return initContainers
		}
	}
	agentVersion := config.DefaultInjectorConfig.AgentConfig.Version
	if av, ok := targetPod.Labels[config.AgentVersionLabel]; ok {
		if v, ok := config.InjectorAgentVersion[av]; ok {
			_, configExists := config.InjectorConfigMaps[v.ConfigMapName]
			if v.Enable && configExists {
				agentVersion = v.Version
				log.Info("[mutation] injection-pod: Inject the specified version to pod",
					zap.String("pod", targetPod.Name), zap.String("version", agentVersion))
			}
		}
	}
	agentInitContainer := &corev1.Container{
		Name:  config.InitContainerName,
		Image: config.DefaultInjectorConfig.AgentConfig.Image + ":" + agentVersion,
		//Command:      strings.Split(conf.InitContainerCmd, ","),
		VolumeMounts: createAgentVolumeMounts(config.InitEmptyDirMountPath, deploymentName),
		Env: func(envMap map[string]string) []corev1.EnvVar {
			envVars := make([]corev1.EnvVar, 0, len(envMap))
			for key, value := range envMap {
				envVars = append(envVars, corev1.EnvVar{Name: key, Value: value})
			}
			return envVars
		}(config.DefaultInjectorConfig.AgentConfig.Envs),
	}
	quantityLimitsCPU, _ := k8sresource.ParseQuantity(DefaultCPU)
	quantityLimitsMemory, _ := k8sresource.ParseQuantity(DefaultMemory)
	quantityRequestsCPU, _ := k8sresource.ParseQuantity(DefaultCPU)
	quantityRequestsMemory, _ := k8sresource.ParseQuantity(DefaultMemory)
	agentInitContainer.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    quantityLimitsCPU,
			corev1.ResourceMemory: quantityLimitsMemory,
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    quantityRequestsCPU,
			corev1.ResourceMemory: quantityRequestsMemory,
		},
	}
	cmds := strings.Split(config.InitContainerCmd, ",")
	agentInitContainer.Command = make([]string, 0)
	for _, cmd := range cmds {
		agentInitContainer.Command = append(agentInitContainer.Command, cmd)
	}
	args := strings.Split(config.InitContainerArgs, ",")
	agentInitContainer.Args = make([]string, 0)
	for _, arg := range args {
		agentInitContainer.Args = append(agentInitContainer.Args, arg)
	}

	initContainers = append(initContainers, *agentInitContainer)
	return initContainers
}

func modifyPodContainer(targetPod *corev1.Pod, envs []corev1.EnvVar, deploymentName string) []corev1.Container {
	log.Debugf("[mutation] /injection-pod: the envs is %v\n size is %d\n", envs, len(envs))
	containers := make([]corev1.Container, 0)
	for i, container := range targetPod.Spec.Containers {
		log.Debugf("[mutation] /injection-pod: the container index is %d, container name is %s", i, container.Name)
		if container.Name != "" {
			envMap := make(map[string]corev1.EnvVar)
			if len(container.Env) != 0 {
				envs = append(envs, container.Env...)
			}
			for _, env := range envs {
				envMap[env.Name] = env
				//if envVar, exist := envMap[env.Name]; exist {
				//	envVar.Value = envVar.Value + env.Value
				//} else {
				//	envMap[env.Name] = env
				//}
			}

			func(envMapConfig map[string]string) {
				for key, value := range envMapConfig {
					if envVar, exist := envMap[key]; exist {
						envVar.Value = envVar.Value + " " + value
					} else {
						envMap[key] = corev1.EnvVar{Name: key, Value: value}
					}
				}
			}(config.DefaultInjectorConfig.AgentConfig.Envs)

			mergeEnvs := make([]corev1.EnvVar, 0)
			for _, envVar := range envMap {
				mergeEnvs = append(mergeEnvs, envVar)
			}
			container.Env = mergeEnvs
			log.Debugf("[mutation] /injection-pod: envs = %v", container.Env)

			// add volumeMounts
			agentVolumeMounts := container.VolumeMounts
			addVolumeForContainer := true
			for _, volume := range agentVolumeMounts {
				if volume.Name == AgentVolumeName {
					log.Warnf("[mutation] /injection-pod: A volume [%s] already exists, skipping the addition logic.", AgentVolumeName)
					addVolumeForContainer = false
					break
				}
			}
			if addVolumeForContainer {
				container.VolumeMounts = append(agentVolumeMounts, createAgentVolumeMounts(config.EmptyDirMountPath, deploymentName)...)
				log.Debugf("[mutation] /injection-pod: volumes = %v", container.VolumeMounts)
			}
			containers = append(containers, container)
		}
	}
	return containers
}

func createAgentVolumeMounts(agentVolumePath, deploymentName string) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      AgentVolumeName,
			MountPath: agentVolumePath,
		},
		{
			Name:      deploymentName + "-live-configmap",
			MountPath: config.ConfigMountPath + "/" + config.AgentConfig,
			SubPath:   config.AgentConfig,
		},
		{
			Name:      deploymentName + "-live-configmap",
			MountPath: config.ConfigMountPath + "/" + config.BootConfig,
			SubPath:   config.BootConfig,
		},
		{
			Name:      deploymentName + "-live-configmap",
			MountPath: config.ConfigMountPath + "/" + config.LogConfig,
			SubPath:   config.LogConfig,
		},
	}
}

func addPodVolume(targetPod *corev1.Pod, deploymentName string) []corev1.Volume {
	// add volume
	volumes := targetPod.Spec.Volumes
	for _, volume := range volumes {
		if volume.Name == AgentVolumeName {
			log.Warnf("[mutation] /injection-pod: A volume [%s] already exists, skipping the addition logic.", AgentVolumeName)
			return volumes
		}
	}
	agentVolumes := []corev1.Volume{
		{
			Name: AgentVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: deploymentName + "-live-configmap",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: deploymentName + "-live-configmap",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  config.AgentConfig,
							Path: config.AgentConfig,
						},
						{
							Key:  config.BootConfig,
							Path: config.BootConfig,
						},
						{
							Key:  config.LogConfig,
							Path: config.LogConfig,
						},
					},
				},
			},
		},
	}
	return append(volumes, agentVolumes...)
}

func createPodPatch(target, original *corev1.Pod) ([]byte, error) {
	targetJSON, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	originalJSON, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.CreatePatch(originalJSON, targetJSON)
	if err != nil {
		return nil, err
	}
	return json.Marshal(patch)
}

func matchRule(labels map[string]string) (map[string]string, map[string]string) {
	mergedEnvs := make(map[string]string)
	mergedLabels := make(map[string]string)
	for _, rule := range config.InjectorRules {
		if isMatch(rule.MatchLabels, labels) {
			// Merge Envs
			for k, v := range rule.Envs {
				mergedEnvs[k] = v
			}
			// Merge Labels
			for k, v := range rule.Labels {
				mergedLabels[k] = v
			}
		}
	}
	return mergedEnvs, mergedLabels
}

func isMatch(ruleLabels, labels map[string]string) bool {
	for k, v := range ruleLabels {
		if labels[k] != v {
			return false
		}
	}
	return true
}
