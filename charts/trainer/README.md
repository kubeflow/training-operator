# trainer

![Version: 2.0.0](https://img.shields.io/badge/Version-2.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.0.0](https://img.shields.io/badge/AppVersion-2.0.0-informational?style=flat-square)

A Helm chart for deploying Kubeflow trainer on Kubernetes.

**Homepage:** <https://github.com/kubeflow/trainer>

## Introduction

This chart bootstraps a [Kubernetes Trainer](https://github.com/kubeflow/trainer) deployment using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Helm >= 3
- Kubernetes >= 1.20

## Usage

### Add Helm Repo

```bash
helm repo add trainer https://kubeflow.github.io/trainer

helm repo update
```

See [helm repo](https://helm.sh/docs/helm/helm_repo) for command documentation.

### Install the chart

```bash
helm install [RELEASE_NAME] trainer/trainer
```

For example, if you want to create a release with name `trainer` in the `kubeflow-system` namespace:

```shell
helm install trainer trainer/trainer \
    --namespace kubeflow-system \
    --create-namespace
```

Note that by passing the `--create-namespace` flag to the `helm install` command, `helm` will create the release namespace if it does not exist.

See [helm install](https://helm.sh/docs/helm/helm_install) for command documentation.

### Upgrade the chart

```shell
helm upgrade [RELEASE_NAME] trainer/trainer [flags]
```

See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade) for command documentation.

### Uninstall the chart

```shell
helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes resources associated with the chart and deletes the release, except for the `crds`, those will have to be removed manually.

See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall) for command documentation.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| nameOverride | string | `""` | String to partially override release name. |
| fullnameOverride | string | `""` | String to fully override release name. |
| commonLabels | object | `{}` | Common labels to add to the resources. |
| image.registry | string | `"docker.io"` | Image registry. |
| image.repository | string | `"kubeflow/trainer-controller-controller"` | Image repository. |
| image.tag | string | If not set, the chart appVersion will be used. | Image tag. |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy. |
| image.pullSecrets | list | `[]` | Image pull secrets for private image registry. |
| controller.replicas | int | `1` | Number of replicas of controller. |
| controller.labels | object | `{}` | Extra labels for controller pods. |
| controller.annotations | object | `{}` | Extra annotations for controller pods. |
| controller.volumes | list | `[]` | Volumes for controller pods. |
| controller.nodeSelector | object | `{}` | Node selector for controller pods. |
| controller.affinity | object | `{}` | Affinity for controller pods. |
| controller.tolerations | list | `[]` | List of node taints to tolerate for controller pods. |
| controller.env | list | `[]` | Environment variables for controller containers. |
| controller.envFrom | list | `[]` | Environment variable sources for controller containers. |
| controller.volumeMounts | list | `[]` | Volume mounts for controller containers. |
| controller.resources | object | `{}` | Pod resource requests and limits for controller containers. |
| controller.securityContext | object | `{}` | Security context for controller containers. |
| controller.serviceAccount.create | bool | `true` | Specifies whether to create a service account for the controller. |
| controller.serviceAccount.name | string | `""` | Optional name for the controller service account. |
| controller.serviceAccount.annotations | object | `{}` | Extra annotations for the controller service account. |
| controller.serviceAccount.automountServiceAccountToken | bool | `true` | Auto-mount service account token to the controller pods. |
| webhook.enable | bool | `true` | Specifies whether to enable webhook. |
| webhook.failurePolicy | string | `"Fail"` | Specifies how unrecognized errors are handled. Available options are `Ignore` or `Fail`. |
| runtime.preTraining.torchDistributed.enable | bool | `true` |  |
| runtime.preTraining.torchDistributed.image.registry | string | `"docker.io"` |  |
| runtime.preTraining.torchDistributed.image.repository | string | `"pytorch/pytorch"` |  |
| runtime.preTraining.torchDistributed.image.tag | string | `"2.5.0-cuda12.4-cudnn9-runtime"` |  |

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| ChenYi015 | <github@chenyicn.net> | <https://github.com/ChenYi015> |
