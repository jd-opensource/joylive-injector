package resource

import (
	"context"
	"errors"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sync"
)

var globalRes *Resource
var initOnce sync.Once

type Resource struct {
	RestConfig *rest.Config
	ClientSet  *kubernetes.Clientset
}

func initResource() *Resource {
	initOnce.Do(func() {
		var err error
		globalRes = &Resource{}
		var kubeconfig string
		var config *rest.Config

		kubeconfig = os.Getenv("KUBECONFIG")

		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}

		if kubeconfig != "" {
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				log.Warn("failed to build restConfig from kubeconfig: ", zap.Error(err))
			}
		}

		if config == nil {
			config, err = rest.InClusterConfig()
			if err != nil {
				log.Fatalf("init k8s config error: %v", err)
			}
		}

		globalRes.RestConfig = config
		globalRes.ClientSet, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("init k8s client error: %v", err)
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
	if err != nil && !errors2.IsNotFound(err) {
		return err
	}
	if cm == nil || errors2.IsNotFound(err) {
		// create
		log.Debug("create configMap", zap.String("name", configMap.Name), zap.String("namespace", namespace))
		_, err = r.ClientSet.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		log.Debug("update configMap", zap.String("name", configMap.Name), zap.String("namespace", namespace), zap.Any("data", configMap.Data))
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

func (r *Resource) GetNodes() (*corev1.NodeList, error) {
	nodeList, err := r.ClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList, nil
}
