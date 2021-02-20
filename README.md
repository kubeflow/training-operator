# K8s Custom Resource and Operator For TensorFlow jobs

[![Build Status](https://travis-ci.org/kubeflow/tf-operator.svg?branch=master)](https://travis-ci.org/kubeflow/tf-operator)
[![Coverage Status](https://coveralls.io/repos/github/kubeflow/tf-operator/badge.svg?branch=master)](https://coveralls.io/github/kubeflow/tf-operator?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeflow/tf-operator)](https://goreportcard.com/report/github.com/kubeflow/tf-operator)

## Overview

TFJob provides a Kubernetes custom resource that makes it easy to
run distributed or non-distributed TensorFlow jobs on Kubernetes.

- For a complete reference of the custom resource definitions, please refer to the [API Definition](pkg/apis/tensorflow/v1/types.go).
- For details on its design, please refer to the [v1alpha2 design doc](https://github.com/kubeflow/community/blob/master/proposals/tf-operator-design-v1alpha2.md).
- For details on its obersibility, please refer to the [monitoring design doc](docs/monitoring/README.md).

## Prerequisites

* Version >= 1.16 of Kubernetes

## Installation

```bash
kubectl apply -f ./deploy/v1/tf-operator.yaml
```

## Quick Start

Please refer to the [quick-start-v1.md](docs/quick-start-v1.md) and [Kubeflow user guide](https://www.kubeflow.org/docs/guides/components/tftraining/) for more information.

## API Documentation

Please refer to [API Documentation](docs/api/generated.asciidoc)

## Community

You can:

- Join our [Slack](https://join.slack.com/t/kubeflow/shared_invite/zt-lhkwrmkh-JPT2g9eva1oPkS00~VHZDQ) channel.
- Check out [who is using this operator](./docs/adopters.md).

This is a part of Kubeflow, so please see [readme in kubeflow/kubeflow](https://github.com/kubeflow/kubeflow#get-involved) to get in touch with the community.

## Contributing

Please refer to the [developer_guide](developer_guide.md)

## Change Log

Please refer to [CHANGELOG](CHANGELOG.md)

## Version Matrix

The following table lists the most recent few versions of the operator.

| Operator Version | API Version | Kubernetes Version |
| ------------- | ------------- | ------------- |
| `latest` (master HEAD) | `v1` | 1.16+ |
| `v1.0.x`| `v1` | 1.16+ |
