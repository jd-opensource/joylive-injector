repo ?= ghcr.io/jd-opensource
version:=1.0.0-$(shell git rev-parse --short HEAD)-$(shell date +%Y%m%d%H%M%S)

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build image push

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o bin/joylive-injector main.go

image:
	DOCKER_SCAN_SUGGEST=false
	docker build --platform linux/amd64 -t $(repo)/joylive-injector:$(version)-AMD64 -f LocalBuild.dockerfile .

build-image:
	DOCKER_SCAN_SUGGEST=false
	docker build -t $(repo)/joylive-injector:$(version)-AMD64 .

push:
	docker push $(repo)/joylive-injector:$(version)-AMD64

build-charts-crs:
	helm template joylive-injector deploy/joylive-injector > deploy/all-cr.yaml
	helm package deploy/joylive-injector --destination deploy/packages
