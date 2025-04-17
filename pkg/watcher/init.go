package watcher

import (
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	"go.uber.org/zap"
)

func init() {
	// Start the ConfigMap listener and initialize the content
	cmWatcher := NewConfigMapWatcher(resource.GetResource().ClientSet)
	err := cmWatcher.Start()
	if err != nil {
		log.Fatal("start cmWatcher error", zap.Error(err))
	}
	err = cmWatcher.InitConfigMap(config.GetNamespace())
	if err != nil {
		log.Fatal("init cm error", zap.Error(err))
	}

	// Start the AgentVersion listener and initialize the content
	avWatcher := NewAgentVersionWatcher(resource.GetResource().RestConfig)
	err = avWatcher.Start()
	if err != nil {
		log.Fatal("start avWatcher error", zap.Error(err))
	}
	err = avWatcher.InitAgentVersion(config.GetNamespace())
	if err != nil {
		log.Fatal("init agentVersion error", zap.Error(err))
	}
}
