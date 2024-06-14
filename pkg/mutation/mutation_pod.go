package mutation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	AgentVolumeName       = "agent-empty-dir"
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
		patchStr, err := createPatch(targetPod, &pod)
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
	anoEnv := pod.Annotations
	envs := make([]corev1.EnvVar, 0)
	if _, ok := anoEnv[config.MatchLabel]; !ok {
		log.Warnf("[mutation] /injection-pod: the annotations do not have %s", config.MatchLabel)
		return envs
	}
	log.Debugf("[mutation] /injection-pod: the annotations is %s", anoEnv[config.MatchLabel])
	anoEnvMap := make(map[string]string)
	err := json.Unmarshal([]byte(anoEnv[config.MatchLabel]), &anoEnvMap)
	if err != nil {
		log.Errorf("[mutation] /injection-pod: failed to unmarshal annotations: %v", err)
		return envs
	}
	for k, v := range anoEnvMap {
		if k != "" && v != "" {
			envs = append(envs, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	return envs
}

func addPodInitContainer(targetPod *corev1.Pod, envs []corev1.EnvVar, deploymentName string) []corev1.Container {
	initContainers := targetPod.Spec.InitContainers
	for _, container := range initContainers {
		if container.Name == config.InitContainerName {
			log.Warnf("[mutation] /injection-pod: A container [%s] already exists, skipping the addition logic.", config.InitContainerName)
			return initContainers
		}
	}
	addVolumes := []corev1.VolumeMount{
		{
			Name:      AgentVolumeName,
			MountPath: config.InitEmptyDirMountPath,
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
	agentVersion := config.InjectorConfig.AgentConfig.Version
	if av, ok := targetPod.Labels[config.AgentVersionLabel]; ok {
		agentVersion = av
	}
	agentInitContainer := &corev1.Container{
		Name:  config.InitContainerName,
		Image: config.InjectorConfig.AgentConfig.Image + ":" + agentVersion,
		//Command:      strings.Split(conf.InitContainerCmd, ","),
		VolumeMounts: addVolumes,
		Env: func(envMap map[string]string) []corev1.EnvVar {
			envVars := make([]corev1.EnvVar, 0, len(envMap))
			for key, value := range envMap {
				envVars = append(envVars, corev1.EnvVar{Name: key, Value: value})
			}
			return envVars
		}(config.InjectorConfig.AgentConfig.Env),
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
			}

			func(envMapConfig map[string]string) {
				for key, value := range envMapConfig {
					envMap[key] = corev1.EnvVar{Name: key, Value: value}
				}
			}(config.InjectorConfig.AgentConfig.Env)

			mergeEnvs := make([]corev1.EnvVar, 0)
			for _, envVar := range envMap {
				mergeEnvs = append(mergeEnvs, envVar)
			}
			container.Env = mergeEnvs
			log.Debugf("[mutation] /injection-pod: envs = %v", container.Env)

			// add volumeMounts
			volumesConfig := []corev1.VolumeMount{
				{
					Name:      AgentVolumeName,
					MountPath: config.EmptyDirMountPath,
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
				container.VolumeMounts = append(agentVolumeMounts, volumesConfig...)
				log.Debugf("[mutation] /injection-pod: volumes = %v", container.VolumeMounts)
			}
			containers = append(containers, container)
		}
	}
	return containers
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

func createPatch(target *corev1.Pod, original *corev1.Pod) ([]byte, error) {
	targetPod, err := json.Marshal(target)
	originalPod, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	p, err := jsonpatch.CreatePatch(originalPod, targetPod)
	if err != nil {
		return nil, err
	}
	return json.Marshal(p)
}
