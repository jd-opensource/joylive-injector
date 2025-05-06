package apm

import (
	"context"
	v1 "k8s.io/api/apps/v1"
	"sync"
)

type Appender interface {
	Modify(ctx context.Context, target *v1.Deployment) (bool, error)
}

// AppenderTypes is a factory holder for any AppenderType
var AppenderTypes = make(map[string]Appender)

var mux sync.Mutex

// RegisterAppenderType Inject the Appender factory to automatically generate Appender objects.
func RegisterAppenderType(key string, appender Appender) {
	mux.Lock()
	defer mux.Unlock()
	if _, ok := AppenderTypes[key]; !ok {
		AppenderTypes[key] = appender
	}
}
