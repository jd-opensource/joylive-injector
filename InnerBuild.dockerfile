ARG BUILD_IMAGE=golang:alpine
ARG RUNTIME_IMAGE=alpine

FROM ${BUILD_IMAGE} AS builder

WORKDIR /workspace

COPY . .

ENV GO111MODULE=on
ENV GOPROXY http://jdos-goproxy.jdcloud.com,direct
ENV GOPRIVATE ""

ARG APP=joylive-injector
ARG RELEASE_TAG=$(VERSION)

RUN go env

# Build the application
RUN go build -ldflags "-w -s -X main.VERSION=${RELEASE_TAG}" -o ./${APP} .

FROM ${RUNTIME_IMAGE}

ARG TZ="Asia/Shanghai"

ENV TZ ${TZ}
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8
ENV LANGUAGE en_US:en

RUN set -ex \
    && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && rm -rf /var/cache/apk/*

COPY --from=builder /workspace/joylive-injector /joylive-injector

ENTRYPOINT ["/joylive-injector"]
