# joylive-injector

[![GitHub repo](https://img.shields.io/badge/GitHub-repo-blue)](https://github.com/jd-opensource/joylive-injector)
[![GitHub release](https://img.shields.io/github/release/jd-opensource/joylive-injector.svg)](https://github.com/jd-opensource/joylive-injector/releases)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://joylivehq.slack.com)

[English](./README.md) | 简体中文

## 介绍
这是一个针对kubernetes的动态准入控制webhook，它可以用于修改`kubernete`资源。
此程序监视`deployments`的CREATE、UPDATE、DELETE事件和`pods`的CREATE事件，并为`POD`添加initContainer、默认增加环境变量`JAVA_TOOL_OPTIONS`、挂载configmap、修改主容器的卷装载等操作。

## 特性
- 支持自动将`joylive-agent`注入Java应用的Pod。
- 支持多版本`joylive-agent`与对应配置管理。
- 支持注入指定版本`joylive-agent`及对应配置。

## 使用方式

因证书签名已按照命名空间为`joylive`预生成，所以须指定安装到对应命名空间。 执行命令：
```bash
helm repo add joylive https://jd-opensource.github.io/joylive-helm-charts
helm repo update
kubectl create namespace joylive
helm install joylive-injector joylive/joylive-injector -n joylive
```
