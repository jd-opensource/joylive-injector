# joylive-injector

[![GitHub repo](https://img.shields.io/badge/GitHub-repo-blue)](https://github.com/jd-opensource/joylive-injector)
[![GitHub release](https://img.shields.io/github/release/jd-opensource/joylive-injector.svg)](https://github.com/jd-opensource/joylive-injector/releases)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://joylivehq.slack.com)

English | [简体中文](./README-zh.md)

## Description
This is a dynamic admission control webhook for kubernetes, it can be used to mutate kubernetes resources.
This program monitors the `CREATE`, `UPDATE`, `DELETE` events for `deployments` and the `CREATE` events for `pods` and adds the initContainer for `Pod` , adds the environment variable `JAVA_TOOL_OPTIONS` by default, mounts the configmap, modifies the volume load for the main container, and so on.

## Features
- Supports automatically injecting `joylive-agent` into Pods of Java applications.
- Supports multi-version `joylive-agent` and corresponding configuration management.
- Support injection of specified version `joylive-agent` and corresponding configuration.

## Used
Since the certificate signature has been pre-generated according to the namespace `joylive`, it is necessary to specify installation to the corresponding namespace. Execute the command:
```bash
helm repo add joylive https://jd-opensource.github.io/joylive-helm-charts
helm repo update
kubectl create namespace joylive
helm install joylive-injector joylive/joylive-injector -n joylive
```
