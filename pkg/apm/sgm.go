package apm

import (
	"context"
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"k8s.io/api/apps/v1"
)

type SgmAppender struct {
}

// SgmAppender implements the Appender interface
var _ Appender = &SgmAppender{}

// init registers the SgmAppender with the Appender factory
func init() {
	RegisterAppenderType("sgm", &SgmAppender{})
}

// Modify modifies the Deployment object by adding specific labels to its template
func (s *SgmAppender) Modify(ctx context.Context, target *v1.Deployment) (bool, error) {
	added := false
	if _, ok := target.Spec.Template.Labels["sgm.jd.com/app"]; !ok {
		target.Spec.Template.Labels["sgm.jd.com/app"] = target.Labels[config.ApplicationLabel]
		added = true
	}
	if _, ok := target.Spec.Template.Labels["sgm.jd.com/group"]; !ok {
		target.Spec.Template.Labels["sgm.jd.com/group"] = target.Labels[config.ServiceGroupLabel]
		added = true
	}
	if _, ok := target.Spec.Template.Labels["sgm.jd.com/probe-inject"]; !ok {
		target.Spec.Template.Labels["sgm.jd.com/probe-inject"] = "true"
		added = true
	}
	if _, ok := target.Spec.Template.Labels["sgm.jd.com/sink"]; !ok {
		target.Spec.Template.Labels["sgm.jd.com/sink"] = "Http"
		added = true
	}
	if _, ok := target.Spec.Template.Labels["sgm.jd.com/tenant"]; !ok {
		target.Spec.Template.Labels["sgm.jd.com/tenant"] = target.Labels[config.TenantLabel]
		added = true
	}
	return added, nil
}
