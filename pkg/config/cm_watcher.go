package config

import (
	"context"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type ConfigMapWatcher struct {
	informerFactory   informers.SharedInformerFactory
	configMapInformer cache.SharedIndexInformer
	configMapLister   listerv1.ConfigMapLister
	cmQueue           workqueue.RateLimitingInterface
}

func NewConfigMapWatcher(kubeClient kubernetes.Interface) *ConfigMapWatcher {
	factory := informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		time.Second*10,
		informers.WithNamespace(GetNamespace()),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			//options.LabelSelector = labels.SelectorFromSet(map[string]string{
			//	"owner": "joylive",
			//}).String()
		}),
	)
	return &ConfigMapWatcher{
		informerFactory:   factory,
		configMapInformer: factory.Core().V1().ConfigMaps().Informer(),
		configMapLister:   factory.Core().V1().ConfigMaps().Lister(),
		cmQueue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func (w *ConfigMapWatcher) InitConfigMap(namespace, name string) error {
	cm, err := w.configMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		log.Error("get configMap error", zap.String("namespace", namespace), zap.String("cmName", name), zap.Error(err))
		return err
	}
	err = w.cacheConfigMap(cm)
	if err != nil {
		log.Error("cache configMap error", zap.String("cmName", cm.Name), zap.Error(err))
		return err
	}
	return nil
}

func (w *ConfigMapWatcher) Start() error {
	ctx := context.Background()
	w.informerFactory.Start(ctx.Done())
	res := w.informerFactory.WaitForCacheSync(ctx.Done())
	for name, synced := range res {
		if !synced {
			log.Info("cache for %s is not synced", zap.String("name", name.Name()))
		}
	}
	configMapEventHandlerFunc := func() cache.ResourceEventHandlerFuncs {
		return cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.createOrUpdateCache(obj)
			},
			UpdateFunc: func(_ interface{}, obj interface{}) {
				w.createOrUpdateCache(obj)
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err != nil {
					log.Error("failed on configmap DeleteFunc", zap.Error(err))
					return
				}
				log.Warn("configMap deleted", zap.String("cm", key))
				InjectorConfigMap = make(map[string]string)
			},
		}
	}
	_, err := w.configMapInformer.AddEventHandler(configMapEventHandlerFunc())
	if err != nil {
		return err
	}
	go w.configMapInformer.Run(ctx.Done())
	go func() {
		for w.processConfigMap() {
		}
	}()
	return nil
}

func (w *ConfigMapWatcher) cacheConfigMap(configMap *v1.ConfigMap) error {
	if configMap == nil || configMap.Data == nil {
		return fmt.Errorf("config map not be nil")
	}
	cmDataBytes, err := yaml.Marshal(configMap.Data)
	if err != nil {
		return err
	}
	cmDataString := string(cmDataBytes)
	log.Info("Received ConfigMap update event, start updating local configuration.", zap.String("cm", configMap.Name),
		zap.String("data", cmDataString))
	InjectorConfigMap = configMap.Data
	if data, ok := configMap.Data[AgentInjectConfigName]; ok {
		c, err := GetAgentInjectConfig(data)
		if err != nil {
			return err
		}
		InjectorConfig = c
		delete(InjectorConfigMap, AgentInjectConfigName)
	}
	return nil
}

func (w *ConfigMapWatcher) createOrUpdateCache(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error("failed on createOrUpdateCache", zap.Error(err))
		return
	}
	w.cmQueue.Add(key)
}

func (w *ConfigMapWatcher) processConfigMap() bool {
	key, quit := w.cmQueue.Get()
	if quit {
		log.Info("configMap queue quit", zap.String("key", key.(string)))
		return false
	}
	defer w.cmQueue.Done(key)
	item, exists, err := w.configMapInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Error("get configMap by key error", zap.String("key", key.(string)), zap.Error(err))
		return true
	}
	if !exists {
		log.Info("configMap not exist", zap.String("key", key.(string)))
		return true
	}
	if configMap, ok := item.(*v1.ConfigMap); ok {
		if err := w.cacheConfigMap(configMap); err != nil {
			log.Error("cache this configMap error", zap.String("key", key.(string)), zap.Error(err))
			w.cmQueue.AddRateLimited(item)
			return true
		}
	}
	return true
}
