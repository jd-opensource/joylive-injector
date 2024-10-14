ARG BUILD_IMAGE=golang:alpine
FROM ${BUILD_IMAGE} AS builder

ENV SRC_PATH ${GOPATH}/src/joylive-injector

WORKDIR ${SRC_PATH}

COPY . .

RUN set -ex \
    && export BUILD_VERSION=$(cat version) \
    && export BUILD_DATE=$(date "+%F %T") \
    && export COMMIT_SHA1=$(git rev-parse HEAD) \
    && go mod tidy \
    && go install -trimpath -ldflags \
        "-X 'main.version=${BUILD_VERSION}' \
        -X 'main.buildDate=${BUILD_DATE}' \
        -X 'main.commitID=${COMMIT_SHA1}' \
        -w -s"

ARG RUNTIME_IMAGE=alpine
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

COPY --from=builder /go/bin/joylive-injector /joylive-injector

ENTRYPOINT ["/joylive-injector"]
