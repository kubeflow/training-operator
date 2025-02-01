# KEP-2401: Kubeflow LLM Trainer V2

## Authors

- Shao Wang - [@Electronic-Waste](https://github.com/Electronic-Waste)

Creation date: 2025-01-31

Google doc: http://bit.ly/4gp8JGd

## Overview

This document discusses the design of LLM Trainer for [Kubeflow Training v2](../2170-kubeflow-training-v2/README.md), tracked by [this issue](https://github.com/kubeflow/training-operator/issues/2401).

**We decided to implement a custom Trainer to fine-tune LLMs, which will be supported officially via TrainingRuntimes in Kubeflow upstream**. This will greatly ease the workload of writing fine-tuning scripts, and provide an in-box toolkit to fine-tune the LLMs with custom datasets and models for Data Scientists.

(The following picture comes from the slides of [Andrey and Yuki’s talk in KubeCon NA 2024](https://kccncna2024.sched.com/event/1i7nV?iframe=no), which explains the workflow of LLM fine-tuning very well)

![](./llm-fine-tuning-lifecycle.png)

## Motivation

Fine-tuning LLMs on Kubernetes is challenging for Data Scientists due to the complex Kubernetes configurations, diverse fine-tuning techniques, and different distributed strategies like data and model-parallelism. It’s crucial to hide the complex infrastructure configurations from users, and allow them to gracefully shift among diverse fine-tuning techniques and distributed strategies.

By now, Kubeflow Training V1 has implemented a [Trainer for LLM](../2003-train-api/README.md) based on the HuggingFace library. However, it is **integrated with data and model initialization**, which are separated in Kubeflow Training V2, and has **limited fine-tuning mechanism support**. We need to migrate it to V2 and support more fine-tuning configurations and distributed strategies in accordance with the growing needs in custom LLM fine-tuning.

### Goals

- Introduce LLM Trainer V2 that supports some important PEFT mechanisms(e.g LoRA, QLoRA, AdapterPrompt, PrefixTuning), allows users to specify sharding policies for distributed training like FSDP and ZeRO, and provides multiple fine-tuning frameworks including HuggingFace Transformers, Native PyTorch and Nvidia NeMo.
- Update Kubeflow Training SDK to allow users to pass their custom data preprocessing logic as a parameter and fine-tune LLMs with LLM Trainer V2 flexibly.
- Create community-supported `ClusterTrainingRuntime` for LLM fine-tuning for various foundational models (e.g. Mistral, LLama-70b, Gemma-7b).

### Non-Goals

- Upgrade LLM Trainer V1.
- Support training launchers besides `torchrun` (could be added after the initial implementation).

## Proposal

## Design Details

## Implementation History

- 2025-01-31: Create KEP-2401 doc

## Alternatives
