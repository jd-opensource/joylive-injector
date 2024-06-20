# joylive-injector

[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://joylivehq.slack.com)

<img src="docs/image/weixin.jpg" alt="pic" style="zoom:33%;" />

[English](./README.md) | 简体中文

## 介绍
这是一个针对kubernetes的动态准入控制webhook，它可以用于修改`kubernete`资源。
此程序监视`deployments`的CREATE、UPDATE、DELETE事件和`pods`的CREATE事件，并为`POD`添加initContainer、默认增加环境变量`JAVA_TOOL_OPTIONS`、挂载configmap、修改主容器的卷装载等操作。

## 特性
- 支持自动将`joylive-agent`注入Java应用的Pod。
- 支持多版本`joylive-agent`与对应配置管理。
- 支持注入指定版本`joylive-agent`及对应配置。

## 使用方式
### 完全模式
- 在要部署的环境中安装 CFSSL(用于签名，验证和捆绑TLS证书的HTTP API工具)
    ```bash
    wget https://pkg.cfssl.org/R1.2/cfssl-certinfo_linux-amd64
    wget https://pkg.cfssl.org/R1.2/cfssl_linux-amd64
    wget https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64 
    mv cfssl-certinfo_linux-amd64 /usr/local/bin/cfssl-certinfo 
    mv cfssl_linux-amd64 /usr/local/bin/cfssl
    mv cfssljson_linux-amd64 /usr/local/bin/cfssljson
    chmod +x /usr/local/bin/cfssl-certinfo /usr/local/bin/cfssl /usr/local/bin/cfssljson
    ```
- 拷贝deploy目录下的`cfssl`和`joylive-injector`到要部署的环境
- `cfssl/dac-csr.json`中的namespace目前填写的是`joylive`，需要根据实际情况修改
- 执行`joylive-injector/deploy/cfssl`目录下的`create-secret.sh`脚本生成secret，若`joylive-injector`包与`cfssl`在同一目录下，可自动替换`caBundle`, `caKeyBundle` 和 `caPubBundle`字段的值
- 若`caBundle`，`caKeyBundle` 和 `caPubBundle`的值未替换，需要手动替换chart包中的`value.yaml`中的`caBundle`，`caKeyBundle` 和 `caPubBundle`字段得值，使用`cat dac-ca.pem | base64 | tr -d '\n'` 作为 `caBundle`, `cat dac-key.pem | base64 | tr -d '\n'` 作为 `caKeyBundle`, `cat dac.pem | base64 | tr -d '\n'` 作为 `caPubBundle` 生成的内容替换
- 执行`helm install joylive-injector ./joylive-injector -n joylive`安装webhook
- chart包中的`value.yaml`中配置按需修改

### 简单模式
执行命令：
```bash
helm install joylive-injector ./packages/joylive-injector-1.0.0.tgz 
```