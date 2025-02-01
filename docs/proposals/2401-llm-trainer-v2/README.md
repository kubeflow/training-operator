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

### Native PyTorch Launcher - `torchtune`

`torchtune` is a PyTorch-native library for easily authoring, fine-tuning and experimenting with LLMs. It provides rich support for LLM fine-tuning:

1. Modular native-PyTorch implementations of popular LLMs
2. Training recipes for a variety of fine-tuning techniques
3. Support for distributed training using [FSDP2](https://github.com/pytorch/torchtitan/blob/main/docs/fsdp.md)
4. YAML configs for easily configuring training runs

`torchtune` is something like our LLM Trainer, because its [core concepts](https://pytorch.org/torchtune/main/overview.html#key-concepts) "recipes" and "configs" can be easily corresponded to our “LLM Trainer Script” and “[Trainer field in TrainJob](https://github.com/kubeflow/training-operator/blob/cf741267f8f8ec96592178532b6787bab3f11110/pkg/apis/kubeflow.org/v2alpha1/trainjob_types.go#L110-L111)”. **It’s the easiest way for us to implement the LLM Trainer**.

**However, `torchtune` only supports single-node training**, which means that we can only have 1 pod in the training phase (`--nnodes=1`, [related issue](https://github.com/pytorch/torchtune/issues/2018)). This would put a strong restriction for us on scaling training pods on Kubernetes. And also, **it only supports some popular LLMs** and will bring inflexibility for us to fine-tune other models.

An example for using `torchtune`:

```bash
$ tune ls
RECIPE                                   CONFIG
full_finetune_single_device              llama2/7B_full_low_memory
                                         mistral/7B_full_low_memory
full_finetune_distributed                llama2/7B_full
                                         llama2/13B_full
                                         mistral/7B_full
lora_finetune_single_device              llama2/7B_lora_single_device
                                         llama2/7B_qlora_single_device
                                         mistral/7B_lora_single_device

$ tune run lora_finetune_single_device --config llama2/7B_lora_single_device epochs=1
INFO:torchtune.utils.logging:Running LoRAFinetuneRecipeSingleDevice with resolved config:
Writing logs to /tmp/lora_finetune_output/log_1713194212.txt
INFO:torchtune.utils.logging:Model is initialized with precision torch.bfloat16.
INFO:torchtune.utils.logging:Tokenizer is initialized from file.
INFO:torchtune.utils.logging:Optimizer and loss are initialized.
INFO:torchtune.utils.logging:Loss is initialized.
INFO:torchtune.utils.logging:Dataset and Sampler are initialized.
INFO:torchtune.utils.logging:Learning rate scheduler is initialized.
1|52|Loss: 2.3697006702423096:   0%|▏                     | 52/25880 [00:24<3:55:01,  1.83it/s]
```

(**Note**: We need to create a new plugin for `torchtune`, so that it can fit in the yaml-based fine-tuning configurations. And also we may need to explore how to integrate the recipes provided by `torchtune`.)

### HF Accelerate CLI - `accelerate`

Huggingface Accelerate CLI is a simplified distributed training launch tool, which is **targeted to junior users not familiar with distributed training**. The official slogan for Huggingface Accelerate is “Run your raw PyTorch training script on any kind of device”. There are several advantages to adopt it:

1. In-box support for HuggingFace Transformers and Datasets libraries
2. Set many proper default values for mixed-precision training and device allocation
3. Hide some complex configurations like communication backend & env variables.

**If we decide to support HF Accelerate CLI, we need to implement a new runtime plugin**. And also, users will lose their control over the training process, which is unacceptable if they want to have more flexibility and set up their own configurations in the distributed training phase.

If time permitted, it would be great to build our impacts among junior learners by a simple distributed training backend.

```python
import torch
  import torch.nn.functional as F
  from datasets import load_dataset
+ from accelerate import Accelerator

- device = 'cpu'
+ accelerator = Accelerator()

- model = torch.nn.Transformer().to(device)
+ model = torch.nn.Transformer()
  optimizer = torch.optim.Adam(model.parameters())

  dataset = load_dataset('my_dataset')
  data = torch.utils.data.DataLoader(dataset, shuffle=True)

+ model, optimizer, data = accelerator.prepare(model, optimizer, data)

  model.train()
  for epoch in range(10):
      for source, targets in data:
-         source = source.to(device)
-         targets = targets.to(device)

          optimizer.zero_grad()
          output = model(source)
          loss = F.cross_entropy(output, targets)

-         loss.backward()
+         accelerator.backward(loss)

          optimizer.step()

```

```bash
$ accelerate config                              
-------------------------------------------------------------------------------------------------------------------------------------In which compute environment are you running?
This machine                                                                                                                         
-------------------------------------------------------------------------------------------------------------------------------------Which type of machine are you using?                                                                                                 
multi-CPU                                                                                                                            
How many different machines will you use (use more than 1 for multi-node training)? [1]:                                             
Should distributed operations be checked while running for errors? This can avoid timeout issues but will be slower. [yes/NO]: yes   
Do you want to use Intel PyTorch Extension (IPEX) to speed up training on CPU? [yes/NO]:yes                                          
Do you want accelerate to launch mpirun? [yes/NO]: no                                                                                
Do you wish to optimize your script with torch dynamo?[yes/NO]:yes                                                                   
-------------------------------------------------------------------------------------------------------------------------------------Which dynamo backend would you like to use?                                                                                          
Please select a choice using the arrow or number keys, and selecting with enter                                                      
inductor                                                                                                                             
Do you want to customize the defaults sent to torch.compile? [yes/NO]: yes                                                           
-------------------------------------------------------------------------------------------------------------------------------------Which mode do you want to use?                                                                                                       
default                                                                                                                              
Do you want the fullgraph mode or it is ok to break model into several subgraphs? [yes/NO]: yes                                      
Do you want to enable dynamic shape tracing? [yes/NO]: yes                                                                           
How many processes should be used for distributed training? [1]:1                                                                    
-------------------------------------------------------------------------------------------------------------------------------------Do you wish to use mixed precision?                                                                                                  
fp16                                                                                                                                 
accelerate configuration saved at /home/xxx/.cache/huggingface/accelerate/default_config.yaml

$ accelerate launch {my_script.py}
```

### Backend Design - `torchtune`

#### New Runtime Plugin

As is shown in the [torchtune official document](https://pytorch.org/torchtune/main/tune_cli.html#run-a-recipe) and [source code](https://github.com/pytorch/torchtune/blob/75965d4281b9b76c454630d015221b9933c77bf3/torchtune/_cli/run.py#L113-L118), the distributed training arguments like `--nnodes` and `--nproc_per_node` should be passed ahead of the recipe argument in the command line, and **cannot be passed by the environment variables** in the `PET_XXX` convention. And also, `torchtune` is extremely different from the fine-tuning paradigm of `torchrun` because it is **recipe and config-based**, which may need more mutation operations in the config file. Here is an [example](https://github.com/Electronic-Waste/kubeflow-llm-trainer/blob/main/torchtune-llm-finetuning.yaml).

Thus, we need to implement a new plugin for `torchtune` if we decide to adopt `torchtune` as a launcher for LLM fine-tuning on Kubernetes. And the new plugin should have these abilities:

1. Parse distributed training arguments in TrainJob and TrainingRuntime API, and integrate them with the `tune run` command.
2. Handle overrides in the `torchtune` fine-tuning configuration file.
3. Validate some requirements, such as `--nnodes` should be equal to 1.

\# WIP
