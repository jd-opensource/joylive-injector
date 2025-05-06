package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/admission"
	"github.com/jd-opensource/joylive-injector/pkg/apm"
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	apiv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

func init() {
	admission.Register(admission.AdmissionFunc{
		Type: admission.AdmissionTypeMutating,
		Path: "/injection-deploy",
		Func: injectionDeploy,
	})

}

// injectionDeploy Handling admission webhook requests, mainly used to handle Deployment resources
func injectionDeploy(request *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	switch request.Kind.Kind {
	case "Deployment":
		log.Debugf("[mutation] /injection-deploy: received request: %v,the operition is %s ", request.Resource, request.Operation)
		if request.Operation == apiv1.Operation("DELETE") {
			log.Debugf("[mutation] /injection-deploy: received delete request name is : %s, namespace is %s ", request.Name, request.Namespace)
			err := deleteConfigMap(request.Name, request.Namespace)
			if err != nil {
				errMsg := fmt.Sprintf("[mutation] /injection-deploy: failed to delete configmap: %v", err)
				log.Error(errMsg)
				return &admissionv1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Code:    http.StatusInternalServerError,
						Message: errMsg,
					},
				}, nil
			}
			return &admissionv1.AdmissionResponse{
				Allowed: true,
				Result: &metav1.Status{
					Code:    http.StatusOK,
					Message: "success",
				},
			}, nil
		}
		var deploy appsv1.Deployment
		err := jsoniter.Unmarshal(request.Object.Raw, &deploy)
		if err != nil {
			errMsg := fmt.Sprintf("[mutation] /injection-deploy: failed to unmarshal object: %v", err)
			log.Errorf(errMsg)
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: errMsg,
				},
			}, nil
		}
		log.Infof("[mutation] /injection-deploy: create config map for this deployment: %s, namespace: %s", deploy.Name, deploy.Namespace)
		err = createOrUpdateConfigMap(&deploy)
		if err != nil {
			errMsg := fmt.Sprintf("[mutation] /injection-deploy: failed to create configmap: %v", err)
			log.Error(errMsg)
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Code:    http.StatusInternalServerError,
					Message: errMsg,
				},
			}, nil
		}

		target := deploy.DeepCopy()
		added := false

		if enhanceType, ok := deploy.Labels[config.EnhanceTypeLabel]; ok && enhanceType == config.EnhanceTypeSidecar {
			log.Infof("[mutation] /injection-deploy: add label %s to deployment %s/%s", config.SidecarEnhanceLabel, deploy.Name, deploy.Namespace)
			target.Spec.Template.Labels[config.SidecarEnhanceLabel] = "true"
			added = true
		} else {
			// Check if the x-live-enabled label exists in deploy's spec.template.metadata.labels; if not, add it.
			if _, ok := deploy.Spec.Template.Labels[config.WebHookMatchKey]; !ok {
				log.Infof("[mutation] /injection-deploy: add label %s to deployment %s/%s", config.WebHookMatchKey, deploy.Name, deploy.Namespace)
				target.Spec.Template.Labels[config.WebHookMatchKey] = config.WebHookMatchValue
				added = true
			}
		}

		// Apm labels append
		if apmType, ok := deploy.Labels[config.ApmTypeLabel]; ok {
			if appender, ok := apm.AppenderTypes[apmType]; ok {
				added, err = appender.Modify(context.Background(), target)
				if err != nil {
					errMsg := fmt.Sprintf("[mutation] /injection-deploy: failed to modify deployment: %v", err)
					log.Error(errMsg)
					return &admissionv1.AdmissionResponse{
						Allowed: false,
						Result: &metav1.Status{
							Code:    http.StatusInternalServerError,
							Message: errMsg,
						},
					}, nil
				}
			} else {
				log.Warnf("[mutation] /injection-deploy: unknown apm type %s", apmType)
			}
		}

		if len(config.ControlPlaneUrl) == 0 {
			return &admissionv1.AdmissionResponse{
				UID:     request.UID,
				Allowed: true,
			}, nil
		} else {
			added, err = AddApplicationEnvironments(deploy, target)
			if err != nil {
				errMsg := fmt.Sprintf("[mutation] /injection-deploy: failed to get application environments: %v", err)
				return &admissionv1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Code:    http.StatusInternalServerError,
						Message: errMsg,
					},
				}, err
			}
		}
		if !added {
			log.Infof("[mutation] /injection-deploy: no envs to add to deployment %s/%s", deploy.Name, deploy.Namespace)
			return &admissionv1.AdmissionResponse{
				UID:     request.UID,
				Allowed: true,
			}, nil
		} else {
			patchStr, err := createDeployPatch(target, &deploy)
			if err != nil {
				log.Errorf("[mutation] /injection-deploy: failed to create patch: %v", err)
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
		}
	default:
		return &admissionv1.AdmissionResponse{
			UID:     request.UID,
			Allowed: true,
		}, nil
	}
}

func AddApplicationEnvironments(source appsv1.Deployment, target *appsv1.Deployment) (bool, error) {
	added := false
	// Get the application environment variables
	envs, err := resource.GetApplicationEnvironments(source.GetLabels())
	if err != nil {
		return added, err
	}
	for k, v := range envs {
		// Check if the environment variable already exists
		exists := false
		for _, env := range target.Spec.Template.Spec.Containers[0].Env {
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
		target.Spec.Template.Spec.Containers[0].Env = append(target.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: k, Value: v})
		added = true
	}
	log.Infof("[mutation] /injection-deploy: add envs to deployment %s/%s, envs: %v, deploy's envs: %v",
		source.Name, source.Namespace, envs, target.Spec.Template.Spec.Containers[0].Env)
	return added, nil
}

func createOrUpdateConfigMap(deploy *appsv1.Deployment) error {
	configMapData := config.DefaultInjectorConfigMap
	if version, ok := deploy.Spec.Template.Labels[config.AgentVersionLabel]; ok {
		if agentVersion, ok := config.InjectorAgentVersion[version]; ok {
			cmd, configExists := config.InjectorConfigMaps[agentVersion.ConfigMapName]
			if agentVersion.Enable && configExists {
				configMapData = cmd
				log.Info("[mutation] injection-deploy: Inject the specified version of configMap",
					zap.String("deployment", deploy.Name), zap.String("version", version),
					zap.String("cmName", agentVersion.ConfigMapName))
			}
		}
	}
	rs := resource.GetResource()
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploy.Name + "-live-configmap",
			Namespace: deploy.Namespace,
		},
		Data: configMapData,
	}
	err := rs.CreateOrUpdateConfigMap(context.Background(), deploy.Namespace, configMap)
	if err != nil {
		log.Errorf("create configmap %s in %s error: %v", deploy.Name, deploy.Namespace, err)
		return err
	}
	return nil
}

func deleteConfigMap(name, namespace string) error {
	rs := resource.GetResource()
	err := rs.DeleteConfigMap(context.Background(), namespace, name+"-live-configmap")
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		log.Errorf("delete configmap %s in %s error: %v", name, namespace, err)
		return err
	}
	log.Info("deleted configmap", zap.String("name", name), zap.String("namespace", namespace))
	return nil
}

func createDeployPatch(target, original *appsv1.Deployment) ([]byte, error) {
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
