# KEP-2401: Kubeflow LLM Trainer V2

## Authors

- Shao Wang - [@Electronic-Waste](https://github.com/Electronic-Waste)

Creation date: 2025-01-31

Google doc: http://bit.ly/4gp8JGd

## Overview

This document discusses the design of LLM Trainer for [Kubeflow Training v2](../2170-kubeflow-training-v2/README.md), tracked by [this issue](https://github.com/kubeflow/training-operator/issues/2401).

**We decided to implement a custom Trainer to fine-tune LLMs, which will be supported officially via TrainingRuntimes in Kubeflow upstream**. This will greatly ease the workload of writing fine-tuning scripts for MLOps Engineers, and provide an in-box toolkit to fine-tune the LLMs with custom datasets and models for Data Scientists.

## Motivation

By now, Kubeflow Training V1 has implemented a [Trainer for LLM](../2003-train-api/README.md) based on the HuggingFace library. However, it is **integrated with data and model initialization**, which are separated in Kubeflow Training V2, and has **limited fine-tuning mechanism support**. We need to migrate it to V2 and support more fine-tuning configurations in accordance with the growing needs in custom LLM fine-tuning.

### Goals

### Non-Goals

## Proposal

## Design Details

## Implementation History

- 2025-01-31: Create KEP-2401 doc

## Alternatives