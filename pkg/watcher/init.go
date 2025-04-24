package watcher

import (
	"sync"

	"github.com/jd-opensource/joylive-injector/pkg/config"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"github.com/jd-opensource/joylive-injector/pkg/resource"
	"go.uber.org/zap"
)

func init() {
	var wg sync.WaitGroup
	var fatalErr error
	var once sync.Once

	wg.Add(2)

	rs := resource.GetResource()
	namespace := config.GetNamespace()

	go func() {
		defer wg.Done()
		cmWatcher := NewConfigMapWatcher(rs.ClientSet)
		err := cmWatcher.Start()
		if err != nil {
			once.Do(func() {
				fatalErr = err
			})
			return
		}
		err = cmWatcher.InitConfigMap(namespace)
		if err != nil {
			once.Do(func() {
				fatalErr = err
			})
		}
	}()

	go func() {
		defer wg.Done()
		avWatcher := NewAgentVersionWatcher(rs.RestConfig)
		err := avWatcher.Start()
		if err != nil {
			once.Do(func() {
				fatalErr = err
			})
			return
		}
		err = avWatcher.InitAgentVersion(namespace)
		if err != nil {
			once.Do(func() {
				fatalErr = err
			})
		}
	}()

	wg.Wait()
	if fatalErr != nil {
		log.Fatal("watcher init error", zap.Error(fatalErr))
	}
}
