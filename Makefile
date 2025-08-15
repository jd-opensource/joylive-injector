repo ?= hub.jdcloud.com/jmsf
version:=1.2.4-$(shell git rev-parse --short HEAD)

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: gen-client build image push build-charts-crs

all-image: build image push

gen-client:
	hack/update-codegen.sh

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o bin/joylive-injector main.go

image:
	DOCKER_SCAN_SUGGEST=false
	docker build --platform linux/amd64 -t $(repo)/joylive-injector:$(version)-amd64 -f LocalBuild.dockerfile .

build-image:
	DOCKER_SCAN_SUGGEST=false
	docker build -t $(repo)/joylive-injector:$(version) .

push:
	docker push $(repo)/joylive-injector:$(version)-amd64

build-charts-crs:
	helm template joylive-injector deploy/joylive-injector --include-crds > deploy/all-cr.yaml
	helm package deploy/joylive-injector --destination deploy/packages
