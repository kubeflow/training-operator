# Kubeflow Training Operator

[![Build Status](https://travis-ci.org/kubeflow/tf-operator.svg?branch=master)](https://travis-ci.org/kubeflow/tf-operator)
[![Coverage Status](https://coveralls.io/repos/github/kubeflow/tf-operator/badge.svg?branch=master)](https://coveralls.io/github/kubeflow/tf-operator?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeflow/tf-operator)](https://goreportcard.com/report/github.com/kubeflow/tf-operator)

## Overview

Before v1.2 release, tensorflow-operator can only support TFJob on Kubernetes.
Starting from v1.3, Training Operator provides Kubernetes custom resources that makes it easy to
run distributed or non-distributed TensorFlow/PyTorch/MXNet/XGBoost jobs on Kubernetes.

- For a complete reference of the custom resource definitions, please refer to the API Definition.
  - [Tensorflow API Definition](pkg/apis/tensorflow/v1/types.go)
  - [PyTorch API Definition](pkg/apis/pytorch/v1/types.go)
  - [MXNet API Definition](pkg/apis/mxnet/v1/types.go)
  - [XGBoost API Definition](pkg/apis/xgboost/v1/types.go)
- For details on API design, please refer to the [v1alpha2 design doc](https://github.com/kubeflow/community/blob/master/proposals/tf-operator-design-v1alpha2.md).
- For details of all-in-one operator design, please refer to the [All-in-one Kubeflow Training Operator](https://docs.google.com/document/d/1x1JPDQfDMIbnoQRftDH1IzGU0qvHGSU4W6Jl4rJLPhI/edit#heading=h.e33ufidnl8z6)
- For details on its obersibility, please refer to the [monitoring design doc](docs/monitoring/README.md).

## Prerequisites

* Version >= 1.16 of Kubernetes
* Version >= 3.x of Kustomize
* Version >= 1.21.x of Kubectl

## Installation

### Master Branch

```bash
kubectl apply -k "github.com/kubeflow/tf-operator.git/manifests/overlays/standalone?ref=master"
```

### Specific Release

```bash
kubectl apply -k "github.com/kubeflow/tf-operator.git/manifests/overlays/standalone?ref=v1.3.0"
```

### Tensorflow Release Only

For users who prefer to use original tensorflow controllers, please checkout v1.2-branch, we will maintain the bug fix in this branch.

```bash
kubectl apply -k "github.com/kubeflow/tf-operator.git/manifests/overlays/standalone?ref=v1.2.0"
```

## Quick Start

Please refer to the [quick-start-v1.md](docs/quick-start-v1.md) and [Kubeflow Training User Guide](https://www.kubeflow.org/docs/guides/components/tftraining/) for more information.

## API Documentation

Please refer to API Documentation.
- [Tensorflow API Documentation](docs/api/tensorflow_generated.asciidoc)
- [PyTorch API Documentation](docs/api/pytorch_generated.asciidoc)
- [MXNet API Documentation](docs/api/mxnet_generated.asciidoc)
- [XGBoost API Documentation](docs/api/xgboost_generated.asciidoc)

## Community

You can:

- Join our [Slack](https://join.slack.com/t/kubeflow/shared_invite/zt-n73pfj05-l206djXlXk5qdQKs4o1Zkg) channel.
- Check out [who is using this operator](./docs/adopters.md).

This is a part of Kubeflow, so please see [readme in kubeflow/kubeflow](https://github.com/kubeflow/kubeflow#get-involved) to get in touch with the community.

## Contributing

Please refer to the [DEVELOPMENT](docs/development/developer_guide.md)


## Change Log

Please refer to [CHANGELOG](CHANGELOG.md)

## Version Matrix

The following table lists the most recent few versions of the operator.

| Operator Version | API Version | Kubernetes Version |
| ------------- | ------------- | ------------- |
| `v1.0.x`| `v1` | 1.16+ |
| `v1.1.x`| `v1` | 1.16+ |
| `v1.2.x`| `v1` | 1.16+ |
| `v1.3.x`| `v1` | 1.18+ |
| `latest` (master HEAD) | `v1` | 1.18+ |
