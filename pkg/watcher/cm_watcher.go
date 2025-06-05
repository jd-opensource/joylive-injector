package watcher

import (
	"context"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"os"
	"reflect"
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
		informers.WithNamespace(config.GetNamespace()),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = labels.SelectorFromSet(map[string]string{
				"app": "joylive-injector",
			}).String()
		}),
	)
	return &ConfigMapWatcher{
		informerFactory:   factory,
		configMapInformer: factory.Core().V1().ConfigMaps().Informer(),
		configMapLister:   factory.Core().V1().ConfigMaps().Lister(),
		cmQueue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func (w *ConfigMapWatcher) InitConfigMap(namespace string) error {
	cms, err := w.configMapLister.ConfigMaps(namespace).List(labels.Everything())
	if err != nil {
		log.Error("get configMap error", zap.String("namespace", namespace), zap.Error(err))
		return err
	}
	for _, cm := range cms {
		err = w.cacheConfigMap(cm)
		if err != nil {
			log.Error("cache configMap error", zap.String("cmName", cm.Name), zap.Error(err))
			return err
		}
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
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				if !isConfigMapEqual(oldObj, newObj) {
					w.createOrUpdateCache(newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err != nil {
					log.Error("failed on configmap DeleteFunc", zap.Error(err))
					return
				}
				log.Warn("configMap deleted", zap.String("cm", key))
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

// isConfigMapEqual compares two ConfigMap objects to see if they are equal
func isConfigMapEqual(oldObj, newObj interface{}) bool {
	oldConfigMap, ok1 := oldObj.(*v1.ConfigMap)
	newConfigMap, ok2 := newObj.(*v1.ConfigMap)
	if !ok1 || !ok2 {
		return false
	}
	return reflect.DeepEqual(oldConfigMap.Data, newConfigMap.Data) && reflect.DeepEqual(oldConfigMap.BinaryData, newConfigMap.BinaryData)
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

	defaultConfigMapName := os.Getenv(config.DefaultConfigMapEnvName)
	ruleConfigMapName := os.Getenv(config.RuleConfigMapEnvName)
	switch configMap.Name {
	case defaultConfigMapName:
		config.DefaultInjectorConfigMap = make(map[string]string, len(configMap.Data))
		for k, v := range configMap.Data {
			config.DefaultInjectorConfigMap[k] = v
		}
		if data, ok := configMap.Data[config.InjectorConfigName]; ok {
			c, err := config.GetAgentInjectConfig(data)
			if err != nil {
				return fmt.Errorf("parse injector config error: %w", err)
			}
			config.DefaultInjectorConfig = c
			delete(config.DefaultInjectorConfigMap, config.InjectorConfigName)
		}
	case ruleConfigMapName:
		rules := make(map[string]*config.InjectorRule, len(configMap.Data))
		for key, value := range configMap.Data {
			rule := &config.InjectorRule{}
			if err := yaml.Unmarshal([]byte(value), rule); err != nil {
				return fmt.Errorf("unmarshal rule %s error: %w", key, err)
			}
			rules[key] = rule
		}
		config.InjectorRules = rules
	default:
		if config.InjectorConfigMaps == nil {
			config.InjectorConfigMaps = make(map[string]map[string]string)
		}
		config.InjectorConfigMaps[configMap.Name] = configMap.Data
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
		w.cmQueue.AddRateLimited(key)
		return true
	}

	if !exists {
		log.Info("configMap not exist", zap.String("key", key.(string)))
		return true
	}
	if configMap, ok := item.(*v1.ConfigMap); ok {
		if err := w.cacheConfigMap(configMap); err != nil {
			log.Error("cache this configMap error", zap.String("key", key.(string)), zap.Error(err))
			return true
		}
	}
	return true
}
