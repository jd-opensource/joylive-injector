ARG TPAAS_BUILD_IMAGE
ARG TPAAS_RUNTIME_IMAGE

FROM ${TPAAS_BUILD_IMAGE} as builder

ENV SRC_PATH ${GOPATH}/src/joylive-webhook

WORKDIR ${SRC_PATH}

COPY . .

RUN set -ex \
    #&& yum update -y \
    && yum install -y gcc git \
    && export BUILD_VERSION=$(cat version) \
    && export BUILD_DATE=$(date "+%F %T") \
    && export COMMIT_SHA1=$(git rev-parse HEAD) \
    && go mod tidy \
    && go install -trimpath -ldflags \
        "-X 'main.version=${BUILD_VERSION}' \
        -X 'main.buildDate=${BUILD_DATE}' \
        -X 'main.commitID=${COMMIT_SHA1}' \
        -w -s"

FROM ${TPAAS_RUNTIME_IMAGE}

ARG TZ="Asia/Shanghai"

ENV TZ ${TZ}
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8
ENV LANGUAGE en_US:en

RUN set -ex \
    #&& yum update -y \
    && yum install -y tzdata ca-certificates \
    && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && yum clean all

COPY --from=builder /root/go/bin/joylive-injector /joylive-injector

ENTRYPOINT ["/joylive-injector"]

CMD ["--help"]
