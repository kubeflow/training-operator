# training-operator

![Version: 1.8.1](https://img.shields.io/badge/Version-1.8.1-informational?style=flat-square) ![AppVersion: 1.8.1](https://img.shields.io/badge/AppVersion-1.8.1-informational?style=flat-square)

A Helm chart for deploying Kubeflow Training Operator on Kubernetes.

**Homepage:** <https://github.com/kubeflow/training-operator>

## Introduction

This chart bootstraps a [Kubeflow Training Operator](https://github.com/kubeflow/training-operator) deployment using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Helm >= 3
- Kubernetes >= 1.20

## Usage

### Add Helm Repo

```shell
helm repo add training-operator https://kubeflow.github.io/training-operator

helm repo update
```

See [helm repo](https://helm.sh/docs/helm/helm_repo) for command documentation.

### Install the chart

```shell
helm install training-operator training-operator/training-operator \
    --namespace kubeflow \
    --create-namespace
```

Note that by passing the `--create-namespace` flag to the `helm install` command, `helm` will create the release namespace if it does not exist.

See [helm install](https://helm.sh/docs/helm/helm_install) for command documentation.

### Upgrade the chart

```shell
helm upgrade training-operator training-operator/training-operator [flags]
```

See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade) for command documentation.

### Uninstall the chart

```shell
helm uninstall training-operator --namespace kubeflow
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
| image.repository | string | `"kubeflow/training-operator"` | Image repository. |
| image.tag | string | If not set, the chart appVersion will be used. | Image tag. |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy. |
| image.pullSecrets | list | `[]` | Image pull secrets for private image registry. |
| replicas | int | `1` | Number of replicas of training operator. |
| serviceAccount.create | bool | `true` | Specifies whether to create a service account for the training operator. |
| serviceAccount.name | string | `""` | Optional name for the training operator service account. |
| serviceAccount.annotations | object | `{}` | Extra annotations for the training operator service account. |
| rbac.create | bool | `true` | Specifies whether to create RBAC resources for the training operator. |
| rbac.annotations | object | `{}` | Extra annotations for the training operator RBAC resources. |
| service.type | string | `"ClusterIP"` | Service type. |
| service.port | int | `80` | Service port. |
| labels | object | `{}` | Extra labels for training operator pods. |
| annotations | object | `{}` | Extra annotations for training operator pods. |
| sidecars | list | `[]` | Sidecar containers for training operator pods. |
| volumes | list | `[]` | Volumes for training operator pods. |
| nodeSelector | object | `{}` | Node selector for training operator pods. |
| affinity | object | `{}` | Affinity for training operator pods. |
| tolerations | list | `[]` | List of node taints to tolerate for training operator pods. |
| priorityClassName | string | `""` | Priority class for training operator pods. |
| podSecurityContext | object | `{}` | Security context for training operator pods. |
| env | list | `[]` | Environment variables for training operator containers. |
| envFrom | list | `[]` | Environment variable sources for training operator containers. |
| volumeMounts | list | `[]` | Volume mounts for training operator containers. |
| resources | object | `{}` | Pod resource requests and limits for training operator pods. |
| securityContext | object | `{"allowPrivilegeEscalation":false}` | Security context for training operator containers. |

