package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/admission"
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
		// Add environment variables to the deployment
		if len(config.ControlPlaneUrl) > 0 {
			// Get the application environment variables
			labels := deploy.GetLabels()
			serviceSpace, application := labels[config.ServiceSpaceLabel], labels[config.ApplicationLabel]
			if len(serviceSpace) > 0 && len(application) > 0 {
				envs, err := resource.GetApplicationEnvironments(serviceSpace, application)
				if err != nil {
					errMsg := fmt.Sprintf("[mutation] /injection-deploy: failed to get application environments: %v", err)
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
				for k, v := range envs {
					target.Spec.Template.Spec.Containers[0].Env = append(target.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: k, Value: v})
				}
				log.Infof("[mutation] /injection-deploy: add envs to deployment %s/%s, envs: %v, deploy's envs: %v",
					deploy.Name, deploy.Namespace, envs, target.Spec.Template.Spec.Containers[0].Env)
				patchStr, err := createDeployPatch(target, &deploy)
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
			} else {
				log.Warnf("[mutation] /injection-deploy: the deployment %s/%s does not have the %s or %s label",
					deploy.Name, deploy.Namespace, config.ServiceSpaceLabel, config.ApplicationLabel)
			}
		}
		return &admissionv1.AdmissionResponse{
			UID:     request.UID,
			Allowed: true,
		}, nil
	default:
		return &admissionv1.AdmissionResponse{
			UID:     request.UID,
			Allowed: true,
		}, nil
	}
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
	log.Debug("delete configmap")
	err := rs.DeleteConfigMap(context.Background(), namespace, name+"-live-configmap")
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		log.Errorf("delete configmap %s in %s error: %v", name, namespace, err)
		return err
	}
	return nil
}

func createDeployPatch(target *appsv1.Deployment, original *appsv1.Deployment) ([]byte, error) {
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
