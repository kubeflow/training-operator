# Kubeflow Training Operator

[![Build Status](https://github.com/kubeflow/training-operator/actions/workflows/test-go.yaml/badge.svg?branch=master)](https://github.com/kubeflow/training-operator/actions/workflows/test-go.yaml?branch=master)
[![Coverage Status](https://coveralls.io/repos/github/kubeflow/training-operator/badge.svg?branch=master)](https://coveralls.io/github/kubeflow/training-operator?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeflow/training-operator)](https://goreportcard.com/report/github.com/kubeflow/training-operator)

## Overview

Kubeflow Training Operator is a Kubernetes-native project for fine-tuning and
scalable distributed training of machine learning (ML) models created with various ML frameworks
such as PyTorch, TensorFlow, HuggingFace, [JAX](https://jax.readthedocs.io/en/latest/), DeepSpeed, XGBoost, PaddlePaddle and others.

You can run high-performance computing (HPC) tasks with the Training Operator and `MPIJob` since it
supports running Message Passing Interface (MPI) on Kubernetes which is heavily used for HPC.
The Training Operator implements the V1 API version of MPI Operator. For the MPI Operator V2 version,
please follow [this guide](https://www.kubeflow.org/docs/components/training/user-guides/mpi/) to
install MPI Operator V2.

The Training Operator allows you to use Kubernetes workloads to effectively train your large models
via [Kubernetes Custom Resources APIs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
or using the Training Operator Python SDK.

## Prerequisites

Please check [the official Kubeflow documentation](https://www.kubeflow.org/docs/components/training/installation/#prerequisites)
for prerequisites to install the Training Operator.

## Installation

Please follow [the Kubeflow Training Operator guide](https://www.kubeflow.org/docs/components/training/installation/#installing-the-training-operator)
for the detailed instructions on how to install Training Operator.

### Installing the Control Plane

Run the following command to install the latest stable release of the Training Operator control plane: `v1.8.0`.

```bash
kubectl apply --server-side=true -k "github.com/kubeflow/training-operator.git/manifests/overlays/standalone?ref=v1.8.0"
```

Run the following command to install the latest changes of the Training Operator control plane:

```bash
kubectl apply --server-side=true -k "github.com/kubeflow/training-operator/manifests/overlays/standalone"
```

### Installing the Python SDK

The Training Operator [implements a Python SDK](https://pypi.org/project/kubeflow-training/)
to simplify creation of distributed training and fine-tuning jobs for Data Scientists.

Run the following command to install the latest stable release of the Training SDK:

```
pip install -U kubeflow-training
```

## Getting Started

Please refer to [the getting started guide](https://www.kubeflow.org/docs/components/training/getting-started/#getting-started-with-pytorchjob)
to quickly create your first distributed training job using the Python SDK.

If you want to work directly with Kubernetes Custom Resources provided by Training Operator,
follow [the PyTorchJob MNIST guide](https://www.kubeflow.org/docs/components/training/pytorch/#creating-a-pytorch-training-job).

## Community

The following links provide information on how to get involved in the community:

- Attend [the bi-weekly AutoML and Training Working Group](https://bit.ly/2PWVCkV) community meeting.
- Join our [`#kubeflow-training` Slack channel](https://www.kubeflow.org/docs/about/community/#kubeflow-slack).
- Check out [who is using the Training Operator](ADOPTERS.md).

This is a part of Kubeflow, so please see [readme in kubeflow/kubeflow](https://github.com/kubeflow/kubeflow#get-involved) to get in touch with the community.

## Contributing

Please refer to the [CONTRIBUTING guide](CONTRIBUTING.md).

## Change Log

Please refer to the [CHANGELOG](CHANGELOG.md).

## Version Matrix

The following table lists the most recent few versions of the operator.

| Operator Version       | API Version | Kubernetes Version |
| ---------------------- | ----------- | ------------------ |
| `v1.4.x`               | `v1`        | 1.23+              |
| `v1.5.x`               | `v1`        | 1.23+              |
| `v1.6.x`               | `v1`        | 1.23+              |
| `v1.7.x`               | `v1`        | 1.25+              |
| `v1.8.x`               | `v1`        | 1.27+              |
| `latest` (master HEAD) | `v1`        | 1.27+              |

## Reference

For a complete reference of the custom resource definitions, please refer to the API Definition.

- [TensorFlow API Definition](pkg/apis/kubeflow.org/v1/tensorflow_types.go)
- [PyTorch API Definition](pkg/apis/kubeflow.org/v1/pytorch_types.go)
- [XGBoost API Definition](pkg/apis/kubeflow.org/v1/xgboost_types.go)
- [MPI API Definition](pkg/apis/kubeflow.org/v1/mpi_types.go)
- [PaddlePaddle API Definition](pkg/apis/kubeflow.org/v1/paddlepaddle_types.go)
- [JAX API Definition](pkg/apis/kubeflow.org/v1/jax_types.go)

For details on the Training Operator custom resources APIs, refer to
[the following API documentation](docs/api/kubeflow.org_v1_generated.asciidoc)

## Acknowledgement

This project was originally started as a distributed training operator for TensorFlow and later we
merged efforts from other Kubeflow Training Operators to provide a unified and simplified experience
for both users and developers. We are very grateful to all who filed issues or helped resolve them,
asked and answered questions, and were part of inspiring discussions.
We'd also like to thank everyone who's contributed to and maintained the original operators.

- PyTorch Operator: [list of contributors](https://github.com/kubeflow/pytorch-operator/graphs/contributors)
  and [maintainers](https://github.com/kubeflow/pytorch-operator/blob/master/OWNERS).
- MPI Operator: [list of contributors](https://github.com/kubeflow/mpi-operator/graphs/contributors)
  and [maintainers](https://github.com/kubeflow/mpi-operator/blob/master/OWNERS).
- XGBoost Operator: [list of contributors](https://github.com/kubeflow/xgboost-operator/graphs/contributors)
  and [maintainers](https://github.com/kubeflow/xgboost-operator/blob/master/OWNERS).
- Common library: [list of contributors](https://github.com/kubeflow/common/graphs/contributors) and
  [maintainers](https://github.com/kubeflow/common/blob/master/OWNERS).
