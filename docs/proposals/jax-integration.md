# Kubeflow Enhancement Proposal: Integrate JAX with Kubeflow Training Operator for Distributed Training on Kubernetes

<!-- toc -->
## Table of Contents
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
- [Design Details](#design-details)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubeflow/training-operator/docs/proposals] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubeflow/website/content/en/docs/components/training], for publication to [kubeflow.org]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This Kubeflow Enhancement Proposal (KEP) aims to integrate [JAX](http://jax.readthedocs.io/), a popular machine learning framework, into the Kubeflow Training Operator to enable distributed training and fine-tuning jobs on Kubernetes. This will involve creating a new Kubernetes Custom Resource Definition (CRD) for JAX (JAXJob) and updating the Training Operator controller to support it. The enhancement will also include integrating JAX with the Training Operator Python SDK to provide simple APIs for Data Scientists to create and manage JAXJob on Kubernetes clusters.

## Motivation

JAX has emerged as a popular machine learning framework for high-performance numerical computing and accelerated training on GPUs and TPUs. With its "multi-controller" programming model, JAX is particularly well-suited for distributed training using the Single Program, Multiple Data (SPMD) paradigm. However, running distributed JAX jobs on Kubernetes requires robust orchestration to handle the complexities of multi-process coordination.

Integrating JAX into the Kubeflow Training Operator will simplify distributed JAX training on Kubernetes, providing Data Scientists and ML Engineers with seamless APIs to deploy and manage JAX jobs. This proposal aims to create a new Kubernetes Custom Resource (CR) for JAX, update the Training Operator controller to support it, and provide an intuitive Python SDK for managing JAX jobs.

### Goals

- Develop a new Kubernetes CRD named `JAXJob` for managing JAX distributed training jobs.
- Update the Training Operator to manage `JAXJob` resources.
- Extend the Training Operator Python SDK to support JAX job creation and management.
- Ensure seamless integration with JAX's distributed training API.

### Non-Goals

- Support for non-distributed JAX training jobs (single-process training can continue to use existing mechanisms).
- General-purpose distributed computing support outside of JAX.

## Proposal

### User Stories (Optional)

#### Story 1

As a Data Scientist, I want to use the Kubeflow Training Operator to run distributed JAX training jobs on a Kubernetes cluster so that I can leverage Kubernetes' scalability and resource management capabilities.

#### Story 2

As a Machine Learning Engineer, I want to use a simple Python SDK to define and launch JAX training jobs on Kubernetes, abstracting away the complexity of Kubernetes configurations.

#### Story 3

As a DevOps engineer, I want to manage JAX distributed training jobs using the Kubeflow Training Operator, so I can provide a consistent and scalable infrastructure for machine learning workloads.

### Notes/Constraints/Caveats (Optional)

- Ensuring compatibility with different versions of JAX and Kubernetes.
- Adequate documentation must be provided for users to understand how to configure and run `JAXJob` resources.

## Design Details

### Custom Resource Definition (CRD) for JAXJob

Define a new CRD for JAXJob that includes specifications for:
- Number of processes
- Coordinator address
- Environment variables for JAX distributed training
- Container image and resource requirements
- Job scheduling and retries

#### API (CRD and resulting objects)

##### Custom Resource Definition

The JAXJob CRD will define the schema for JAX distributed training jobs, including specifications for the coordinator, worker processes, and necessary environment variables.

```yaml
---
apiVersion: kubeflow.org/v1
kind: JAXJob
metadata:
  name: example-jaxjob
spec:
  backend: "gloo"
  masterPort: "6666"
  replicaSpecs:
    - replicas: 1
      ReplicaType: MASTER
      template:
        spec:
          containers:
            - image: bitnami/jax:latest
              name: jax
              imagePullPolicy: IfNotPresent
          restartPolicy: OnFailure
    - replicas: 2
      ReplicaType: Worker
      template:
        spec:
          containers:
            - image: bitnami/jax:latest
              name: jax
        restartPolicy: OnFailure

```
Available image options:
1. Bitnami package for JAX: https://hub.docker.com/r/bitnami/jax 
2. NVDIA JAX Toolbox: https://github.com/NVIDIA/JAX-Toolbox/pkgs/container/jax 

This YAML file defines a JAXJob with two worker replicas. Each worker runs a container based on a JAX image and executes a Python script (e.g. train.py). The environment variables necessary for JAX's distributed training are set within each container.

##### Resulting Master

The Master component will act as the coordinator for the distributed JAX processes. It will be responsible for initializing the JAX distributed environment and managing communication between worker processes.

```yaml
---
kind: Service
apiVersion: v1
metadata:
  name: jaxjob-master-${job_id}
spec:
  selector:
    app: jaxjob-master-${job_id}
  ports:
  - port: 6666
    targetPort: 6666
```
```yaml
---
apiVersion: v1
kind: Pod
metadata:
  name: jaxjob-master-${job_id}
  labels:
    app: jaxjobmaster-${job_id}
spec:
  containers:
  - image: bitnami/jax:latest
    name: master
    imagePullPolicy: IfNotPresent
    env:
      - name: JAX_COORDINATOR_ADDRESS
        value: '127.0.0.1:6666'
      - name: JAX_NUM_PROCESSES
        value: 1
      - name: JAX_PROCESS_ID
        value: 0
      - name: MASTER_PORT
        value: "6666"
      - name: MASTER_ADDR
        value: "localhost"
      - name: WORLD_SIZE
        value: "3"
        # Rank 0 is the master
      - name: RANK
        value: "0"
    ports:
      - name: masterPort
        containerPort: 6666
restartPolicy: OnFailure

```

##### Resulting Worker

Worker components will execute the distributed training tasks. Each worker will be assigned a unique process ID and will connect to the coordinator using the specified environment variables.

```yaml
---
apiVersion: v1
kind: Pod
metadata:
  name: jaxjob-worker-${job_id}
spec:
  containers:
    - image: bitnami/jax:latest
      name: worker
      imagePullPolicy: IfNotPresent
      env:
        - name: JAX_COORDINATOR_ADDRESS
          value: '127.0.0.1:6666'
        - name: JAX_NUM_PROCESSES
          value: 1
        - name: JAX_PROCESS_ID
          value: 0
        - name: MASTER_PORT
          value: "23456"
        - name: MASTER_ADDR
          value: pytorch-master-${job_id}
        - name: WORLD_SIZE
          value: "3"
        - name: RANK
          value: "1"
restartPolicy: OnFailure

```
### Controller Update

Update the Training Operator controller to handle JAXJob resources:
- Watch for JAXJob resources and manage their lifecycle.
- Set up the required environment variables for each container in the JAXJob.
- Ensure that JAX distributed training initialization is handled correctly using [jax.distributed.initialize()](https://jax.readthedocs.io/en/latest/_autosummary/jax.distributed.initialize.html).

### Python SDK Integration

Enhance the Training Operator Python SDK to include:
- APIs for defining and launching JAXJob resources.
- Utility functions for setting up distributed training environments.
- Example scripts and templates for common JAX training scenarios.

## Alternatives

- Integrate JAX to Training Operator with JobSet and `TrainJob` after `TrainJob` API is implemented in Training Operator.

## Implementation History

- 2024-05-22: Initial KEP draft created.
