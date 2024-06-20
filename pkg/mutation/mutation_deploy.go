package mutation

import (
	"context"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/admission"
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	jsoniter "github.com/json-iterator/go"
	admissionv1 "k8s.io/api/admission/v1"
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
		log.Debugf("[mutation] ----- /injection-deploy: received request: %v,the operition is %s ", request.Resource, request.Operation)
		if request.Operation == "DELETE" {
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
			log.Error(errMsg)
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: errMsg,
				},
			}, nil
		}
		err = createConfigMap(&deploy)
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

func createConfigMap(deploy *appsv1.Deployment) error {
	configMapData := config.DefaultInjectorConfigMap
	if version, ok := deploy.Spec.Template.Labels[config.AgentVersionLabel]; ok {
		if agentVersion, ok := config.InjectorAgentVersion[version]; ok {
			cmd, configExists := config.InjectorConfigMaps[agentVersion.ConfigMapName]
			if agentVersion.Enable && configExists {
				configMapData = cmd
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
	//logger.Debugf("create configmap: %v", configMap)
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
