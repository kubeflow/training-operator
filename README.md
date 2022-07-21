# Kubeflow Training Operator

[![Build Status](https://github.com/kubeflow/training-operator/actions/workflows/test-go.yaml/badge.svg?branch=master)](https://github.com/kubeflow/training-operator/actions/workflows/test-go.yaml?branch=master)
[![Coverage Status](https://coveralls.io/repos/github/kubeflow/training-operator/badge.svg?branch=master)](https://coveralls.io/github/kubeflow/training-operator?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeflow/training-operator)](https://goreportcard.com/report/github.com/kubeflow/training-operator)

## Overview

Starting from v1.3, this training operator provides Kubernetes custom resources that makes it easy to
run distributed or non-distributed TensorFlow/PyTorch/Apache MXNet/XGBoost/MPI jobs on Kubernetes.

> Note: Before v1.2 release, Kubeflow Training Operator only supports TFJob on Kubernetes.

- For a complete reference of the custom resource definitions, please refer to the API Definition.
  - [TensorFlow API Definition](pkg/apis/kubeflow.org/v1/tensorflow_types.go)
  - [PyTorch API Definition](pkg/apis/kubeflow.org/v1/pytorch_types.go)
  - [Apache MXNet API Definition](pkg/apis/kubeflow.org/v1/mxnet_types.go)
  - [XGBoost API Definition](pkg/apis/kubeflow.org/v1/xgboost_types.go)
  - [MPI API Definition](pkg/apis/kubeflow.org/v1/mpi_types.go)
- For details on API design, please refer to the [v1alpha2 design doc](https://github.com/kubeflow/community/blob/master/proposals/tf-operator-design-v1alpha2.md).
- For details of all-in-one operator design, please refer to the [All-in-one Kubeflow Training Operator](https://docs.google.com/document/d/1x1JPDQfDMIbnoQRftDH1IzGU0qvHGSU4W6Jl4rJLPhI/edit#heading=h.e33ufidnl8z6)
- For details on its observability, please refer to the [monitoring design doc](docs/monitoring/README.md).

## Prerequisites

- Version >= 1.16 of Kubernetes
- Version >= 3.x of Kustomize
- Version >= 1.21.x of Kubectl

## Installation

### Master Branch

```bash
kubectl apply -k "github.com/kubeflow/training-operator/manifests/overlays/standalone"
```

### Stable Release

```bash
kubectl apply -k "github.com/kubeflow/training-operator/manifests/overlays/standalone?ref=v1.3.0"
```

### TensorFlow Release Only

For users who prefer to use original TensorFlow controllers, please checkout `v1.2-branch`, patches for bug fixes will still be accepted to this branch.

```bash
kubectl apply -k "github.com/kubeflow/training-operator/manifests/overlays/standalone?ref=v1.2.0"
```

### Python SDK for Kubeflow Training Operator

Training Operator provides Python SDK for the custom resources. More docs are available in [sdk/python](sdk/python) folder.

Use `pip install` command to install the latest release of the SDK:

```
pip install kubeflow-training
```

## Quick Start

Please refer to the [quick-start-v1.md](docs/quick-start-v1.md) and [Kubeflow Training User Guide](https://www.kubeflow.org/docs/guides/components/tftraining/) for more information.

## API Documentation

Please refer to following API Documentation:

- [Kubeflow.org v1 API Documentation](docs/api/kubeflow.org_v1_generated.asciidoc)

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

| Operator Version       | API Version | Kubernetes Version |
| ---------------------- | ----------- | ------------------ |
| `v1.0.x`               | `v1`        | 1.16+              |
| `v1.1.x`               | `v1`        | 1.16+              |
| `v1.2.x`               | `v1`        | 1.16+              |
| `v1.3.x`               | `v1`        | 1.18+              |
| `latest` (master HEAD) | `v1`        | 1.18+              |

## Acknowledgement

This project was originally started as a distributed training operator for TensorFlow and later we merged efforts from other Kubeflow training operators to provide a unified and simplified experience for both users and developers. We are very grateful to all who filed issues or helped resolve them, asked and answered questions, and were part of inspiring discussions. We'd also like to thank everyone who's contributed to and maintained the original operators.

- PyTorch Operator: [list of contributors](https://github.com/kubeflow/pytorch-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/pytorch-operator/blob/master/OWNERS).
- MPI Operator: [list of contributors](https://github.com/kubeflow/mpi-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/mpi-operator/blob/master/OWNERS).
- XGBoost Operator: [list of contributors](https://github.com/kubeflow/xgboost-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/xgboost-operator/blob/master/OWNERS).
- MXNet Operator: [list of contributors](https://github.com/kubeflow/mxnet-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/mxnet-operator/blob/master/OWNERS).
