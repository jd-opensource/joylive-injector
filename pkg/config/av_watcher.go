package config

import (
	"context"
	"fmt"
	v1 "github.com/jd-opensource/joylive-injector/client-go/apis/injector/v1"
	clientset "github.com/jd-opensource/joylive-injector/client-go/clientset/versioned"
	"github.com/jd-opensource/joylive-injector/client-go/informers/externalversions"
	listerv1 "github.com/jd-opensource/joylive-injector/client-go/listers/injector/v1"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"time"
)

type AgentVersionWatcher struct {
	informerFactory      externalversions.SharedInformerFactory
	agentVersionInformer cache.SharedIndexInformer
	agentVersionLister   listerv1.AgentVersionLister
	cmQueue              workqueue.RateLimitingInterface
}

func NewAgentVersionWatcher(client *rest.Config) *AgentVersionWatcher {
	cs, err := clientset.NewForConfig(client)
	if err != nil {
		log.Fatal("get clientset error", zap.Error(err))
	}
	factory := externalversions.NewSharedInformerFactory(
		cs,
		time.Second*10,
	)
	return &AgentVersionWatcher{
		informerFactory:      factory,
		agentVersionInformer: factory.Injector().V1().AgentVersions().Informer(),
		agentVersionLister:   factory.Injector().V1().AgentVersions().Lister(),
		cmQueue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func (w *AgentVersionWatcher) InitAgentVersion(namespace string) error {
	avs, err := w.agentVersionLister.AgentVersions(namespace).List(labels.Everything())
	if err != nil {
		log.Error("get agentVersions error", zap.String("namespace", namespace), zap.Error(err))
		return err
	}
	for _, av := range avs {
		err = w.cacheAgentVersion(av)
		if err != nil {
			log.Error("cache agentVersion error", zap.String("avName", av.Name), zap.Error(err))
			return err
		}
	}
	return nil
}

func (w *AgentVersionWatcher) Start() error {
	ctx := context.Background()
	w.informerFactory.Start(ctx.Done())
	res := w.informerFactory.WaitForCacheSync(ctx.Done())
	for name, synced := range res {
		if !synced {
			log.Info("cache for %s is not synced", zap.String("name", name.Name()))
		}
	}
	agentVersionEventHandlerFunc := func() cache.ResourceEventHandlerFuncs {
		return cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.createOrUpdateCache(obj)
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				if !isAgentVersionEqual(oldObj, newObj) {
					w.createOrUpdateCache(newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err != nil {
					log.Error("failed on configmap DeleteFunc", zap.Error(err))
					return
				}
				log.Warn("ignore agentVersion deleted", zap.String("cm", key))
			},
		}
	}
	_, err := w.agentVersionInformer.AddEventHandler(agentVersionEventHandlerFunc())
	if err != nil {
		return err
	}
	go w.agentVersionInformer.Run(ctx.Done())
	go func() {
		for w.processAgentVersion() {
		}
	}()
	return nil
}

// isAgentVersionEqual compares two AgentVersion objects to see if they are equal
func isAgentVersionEqual(oldObj, newObj interface{}) bool {
	oldConfigMap, ok1 := oldObj.(*v1.AgentVersion)
	newConfigMap, ok2 := newObj.(*v1.AgentVersion)
	if !ok1 || !ok2 {
		return false
	}
	return reflect.DeepEqual(oldConfigMap.Spec, newConfigMap.Spec)
}

func (w *AgentVersionWatcher) cacheAgentVersion(agentVersion *v1.AgentVersion) error {
	if agentVersion == nil {
		return fmt.Errorf("agentVersion not be nil")
	}
	avSpecBytes, err := yaml.Marshal(agentVersion.Spec)
	if err != nil {
		return err
	}
	avSpec := string(avSpecBytes)
	log.Info("Received AgentVersion update event, start updating local configuration.", zap.String("agentVersion", agentVersion.Name),
		zap.String("agentVersionSpec", avSpec))
	InjectorAgentVersion[agentVersion.Spec.Version] = agentVersion.Spec
	return nil
}

func (w *AgentVersionWatcher) createOrUpdateCache(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error("failed on createOrUpdateCache", zap.Error(err))
		return
	}
	w.cmQueue.Add(key)
}

func (w *AgentVersionWatcher) processAgentVersion() bool {
	key, quit := w.cmQueue.Get()
	if quit {
		log.Info("agentVersion queue quit", zap.String("key", key.(string)))
		return false
	}
	defer w.cmQueue.Done(key)
	item, exists, err := w.agentVersionInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Error("get agentVersion by key error", zap.String("key", key.(string)), zap.Error(err))
		return true
	}
	if !exists {
		log.Info("agentVersion not exist", zap.String("key", key.(string)))
		return true
	}
	if agentVersion, ok := item.(*v1.AgentVersion); ok {
		if err := w.cacheAgentVersion(agentVersion); err != nil {
			log.Error("cache this agentVersion error", zap.String("key", key.(string)), zap.Error(err))
			w.cmQueue.AddRateLimited(item)
			return true
		}
	}
	return true
}
