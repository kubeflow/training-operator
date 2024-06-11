# Kubeflow Training Operator

[![Build Status](https://github.com/kubeflow/training-operator/actions/workflows/test-go.yaml/badge.svg?branch=master)](https://github.com/kubeflow/training-operator/actions/workflows/test-go.yaml?branch=master)
[![Coverage Status](https://coveralls.io/repos/github/kubeflow/training-operator/badge.svg?branch=master)](https://coveralls.io/github/kubeflow/training-operator?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeflow/training-operator)](https://goreportcard.com/report/github.com/kubeflow/training-operator)

## Overview

Kubeflow Training Operator is a Kubernetes-native project for fine-tuning and
scalable distributed training of machine learning (ML) models created with various ML frameworks
such as PyTorch, Tensorflow, XGBoost, MPI, Paddle and others.

Training Operator allows you to use Kubernetes workloads to effectively train your large models
via [Kubernetes Custom Resources APIs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
or using Training Operator Python SDK.

> Note: Before v1.2 release, Kubeflow Training Operator only supports TFJob on Kubernetes.

- For a complete reference of the custom resource definitions, please refer to the API Definition.
  - [TensorFlow API Definition](pkg/apis/kubeflow.org/v1/tensorflow_types.go)
  - [PyTorch API Definition](pkg/apis/kubeflow.org/v1/pytorch_types.go)
  - [Apache MXNet API Definition](pkg/apis/kubeflow.org/v1/mxnet_types.go)
  - [XGBoost API Definition](pkg/apis/kubeflow.org/v1/xgboost_types.go)
  - [MPI API Definition](pkg/apis/kubeflow.org/v1/mpi_types.go)
  - [PaddlePaddle API Definition](pkg/apis/kubeflow.org/v1/paddlepaddle_types.go)
- For details of all-in-one operator design, please refer to the [All-in-one Kubeflow Training Operator](https://docs.google.com/document/d/1x1JPDQfDMIbnoQRftDH1IzGU0qvHGSU4W6Jl4rJLPhI/edit#heading=h.e33ufidnl8z6)
- For details on its observability, please refer to the [monitoring design doc](docs/monitoring/README.md).

## Prerequisites

- Version >= 1.25 of Kubernetes cluster and `kubectl`

## Installation

### Master Branch

```bash
kubectl apply -k "github.com/kubeflow/training-operator/manifests/overlays/standalone"
```

### Stable Release

```bash
kubectl apply -k "github.com/kubeflow/training-operator/manifests/overlays/standalone?ref=v1.7.0"
```

### TensorFlow Release Only

For users who prefer to use original TensorFlow controllers, please checkout `v1.2-branch`, patches for bug fixes will still be accepted to this branch.

```bash
kubectl apply -k "github.com/kubeflow/training-operator/manifests/overlays/standalone?ref=v1.2.0"
```

### Python SDK for Kubeflow Training Operator

Training Operator provides Python SDK for the custom resources. To learn more about available
SDK APIs check [the `TrainingClient`](sdk/python/kubeflow/training/api/training_client.py).

Use `pip install` command to install the latest release of the SDK:

```
pip install kubeflow-training
```

Training Operator controller and Python SDK have the same release versions.

## Quickstart

Please refer to the [getting started guide](https://www.kubeflow.org/docs/components/training/overview/#getting-started)
to quickly create your first Training Operator Job using Python SDK.

If you want to work directly with Kubernetes Custom Resources provided by Training Operator,
follow [the PyTorchJob MNIST guide](https://www.kubeflow.org/docs/components/training/pytorch/#creating-a-pytorch-training-job).

## API Documentation

Please refer to following API Documentation:

- [Kubeflow.org v1 API Documentation](docs/api/kubeflow.org_v1_generated.asciidoc)

## Community

The following links provide information about getting involved in the community:

- Attend [the AutoML and Training Working Group](https://docs.google.com/document/d/1MChKfzrKAeFRtYqypFbMXL6ZIc_OgijjkvbqmwRV-64/edit) community meeting.
- Join our [`#kubeflow-training` Slack channel](https://www.kubeflow.org/docs/about/community/#kubeflow-slack).
- Check out [who is using the Training Operator](./docs/adopters.md).

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
| `v1.4.x`               | `v1`        | 1.23+              |
| `v1.5.x`               | `v1`        | 1.23+              |
| `v1.6.x`               | `v1`        | 1.23+              |
| `v1.7.x`               | `v1`        | 1.25+              |
| `latest` (master HEAD) | `v1`        | 1.25+              |

## Acknowledgement

This project was originally started as a distributed training operator for TensorFlow and later we merged efforts from other Kubeflow training operators to provide a unified and simplified experience for both users and developers. We are very grateful to all who filed issues or helped resolve them, asked and answered questions, and were part of inspiring discussions. We'd also like to thank everyone who's contributed to and maintained the original operators.

- PyTorch Operator: [list of contributors](https://github.com/kubeflow/pytorch-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/pytorch-operator/blob/master/OWNERS).
- MPI Operator: [list of contributors](https://github.com/kubeflow/mpi-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/mpi-operator/blob/master/OWNERS).
- XGBoost Operator: [list of contributors](https://github.com/kubeflow/xgboost-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/xgboost-operator/blob/master/OWNERS).
- MXNet Operator: [list of contributors](https://github.com/kubeflow/mxnet-operator/graphs/contributors) and [maintainers](https://github.com/kubeflow/mxnet-operator/blob/master/OWNERS).
- Common library: [list of contributors](https://github.com/kubeflow/common/graphs/contributors) and [maintainers](https://github.com/kubeflow/common/blob/master/OWNERS).
