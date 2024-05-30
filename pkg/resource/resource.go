package resource

import (
	"context"
	"errors"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sync"
)

var globalRes *Resource
var initOnce sync.Once

type Resource struct {
	ClientSet *kubernetes.Clientset
}

func initResource() *Resource {
	initOnce.Do(func() {
		var err error
		globalRes = &Resource{}
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Errorf("init k8s config error: %v", err)
			panic(err.Error())
		}
		// creates the clientset
		globalRes.ClientSet, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Errorf("init k8s client error: %v", err)
			panic(err.Error())
		}
	})

	return globalRes
}

func GetResource() *Resource {
	if globalRes == nil {
		return initResource()
	}
	return globalRes
}

func (r *Resource) CreateOrUpdateConfigMap(ctx context.Context, namespace string, configMap *corev1.ConfigMap) error {
	cm, err := r.ClientSet.CoreV1().ConfigMaps(namespace).Get(ctx, configMap.Name, metav1.GetOptions{})
	if cm == nil || errors2.IsNotFound(err) {
		// create
		_, err = r.ClientSet.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		cm.Data = configMap.Data
		_, err = r.ClientSet.CoreV1().ConfigMaps(namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Resource) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	err := r.ClientSet.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (r *Resource) GetDeploymentName(pod *corev1.Pod, namespace string) (string, error) {
	// Find ReplicaSet (OwnerReferences) belonging to Deployment
	for _, ownerReference := range pod.OwnerReferences {
		if ownerReference.Kind == "ReplicaSet" {
			// Get ReplicaSet
			rs, err := r.ClientSet.AppsV1().ReplicaSets(namespace).Get(context.TODO(), ownerReference.Name, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			// Find OwnerReferences belonging to Deployment
			for _, rsOwnerReference := range rs.OwnerReferences {
				if rsOwnerReference.Kind == "Deployment" {
					return rsOwnerReference.Name, nil
				}
			}
		}
	}
	return "", errors.New("no corresponding resources found")
}
