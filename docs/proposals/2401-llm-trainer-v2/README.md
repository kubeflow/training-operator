# KEP-2401: Kubeflow LLM Trainer V2

## Authors

- Shao Wang - [@Electronic-Waste](https://github.com/Electronic-Waste)

Creation date: 2025-01-31

Google doc: http://bit.ly/4gp8JGd

## Overview

This document discusses the design of LLM Trainer for [Kubeflow Training v2](../2170-kubeflow-training-v2/README.md), tracked by [this issue](https://github.com/kubeflow/training-operator/issues/2401).

**We decided to implement a custom Trainer to fine-tune LLMs, which will be supported officially via TrainingRuntimes in Kubeflow upstream**. This will greatly ease the workload of writing fine-tuning scripts, and provide an in-box toolkit to fine-tune the LLMs with custom datasets and models for Data Scientists.

## Motivation

Fine-tuning LLMs on Kubernetes is challenging for Data Scientists due to the complex Kubernetes configurations, diverse fine-tuning techniques, and different distributed strategies like data and model-parallelism. It’s crucial to hide the complex infrastructure configurations from users, and allow them to gracefully shift among diverse fine-tuning techniques and distributed strategies.

By now, Kubeflow Training V1 has implemented a [Trainer for LLM](../2003-train-api/README.md) based on the HuggingFace library. However, it is **integrated with data and model initialization**, which are separated in Kubeflow Training V2, and has **limited fine-tuning mechanism support**. We need to migrate it to V2 and support more fine-tuning configurations and distributed strategies in accordance with the growing needs in custom LLM fine-tuning.

### Goals

### Non-Goals

## Proposal

## Design Details

## Implementation History

- 2025-01-31: Create KEP-2401 doc

## Alternatives