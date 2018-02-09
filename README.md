# K8s Custom Resource and Operator For TensorFlow jobs

[![Build Status](https://travis-ci.org/tensorflow/k8s.svg?branch=master)](https://travis-ci.org/tensorflow/k8s)
[![Coverage Status](https://coveralls.io/repos/github/tensorflow/k8s/badge.svg?branch=master)](https://coveralls.io/github/tensorflow/k8s?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/tensorflow/k8s)](https://goreportcard.com/report/github.com/tensorflow/k8s)
[![Prow Test Dashboard](https://img.shields.io/badge/prow-dashboard-blue.svg)](https://k8s-testgrid.appspot.com/sig-big-data)
[![Prow Jobs](https://img.shields.io/badge/prow-status-blue.svg)](https://prow.k8s.io/?repo=tensorflow%2Fk8s)

## Overview

The operator is a part of [Kubeflow](https://github.com/kubeflow/kubeflow)
, which provides a Kubernetes custom resource that makes it easy to
run distributed or non-distributed TensorFlow jobs on Kubernetes.

Using a Custom Resource Definition (CRD) gives users the ability to create and manage TF Jobs just like builtin K8s resources. For example to
create a job

```bash
kubectl create -f examples/tf_job.yaml
```

To list jobs

```bash
kubectl get tfjobs

NAME          KINDS
example-job   TFJob.v1alpha.kubeflow.org
```

## Getting Started

You can easily setup the environment, You could read [the quick start to start a tour of TFJob operator](docs/get_started.md).

## TFJob CRD Design

For additional information about motivation and design for the
CRD please refer to
[TFJob Design Document](docs/tf_job_design_doc.md).

## Contributing

Feel free to hack on tf-operator! We have prepared [a detailed guide](docs/developer_guide.md), which will help you to get involved into the development.

## Community

* [Slack Channel](https://join.slack.com/t/kubeflow/shared_invite/enQtMjgyMzMxNDgyMTQ5LWUwMTIxNmZlZTk2NGU0MmFiNDE4YWJiMzFiOGNkZGZjZmRlNTExNmUwMmQ2NzMwYzk5YzQxOWQyODBlZGY2OTg)
* [Twitter](http://twitter.com/kubeflow)
* [Mailing List](https://groups.google.com/forum/#!forum/kubeflow-discuss)

In the interest of fostering an open and welcoming environment, we as contributors and maintainers pledge to making participation in our project and our community a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, education, socio-economic status, nationality, personal appearance, race, religion, or sexual identity and orientation.

The Kubeflow community is guided by our [Code of Conduct](https://github.com/kubeflow/community/blob/master/CODE_OF_CONDUCT.md), which we encourage everybody to read before participating.
