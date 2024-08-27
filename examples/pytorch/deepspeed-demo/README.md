## Training a Masked Language Model with PyTorch and DeepSpeed

This folder contains an example of training a Masked Language Model with PyTorch and DeepSpeed.

The python script used to train BERT with PyTorch and DeepSpeed. For more information, please refer to the [DeepSpeedExamples](https://github.com/microsoft/DeepSpeedExamples/blob/master/training/HelloDeepSpeed/README.md).

DeepSpeed can be deployed by different launchers such as torchrun, the deepspeed launcher, or Accelerate.
See [deepspeed](https://huggingface.co/docs/transformers/main/en/deepspeed?deploy=multi-GPU&pass-config=path+to+file&multinode=torchrun#deployment).

This guide will show you how to deploy DeepSpeed with the `torchrun` launcher.
The simplest way to quickly reproduce the following is to switch to the DeepSpeedExamples commit:
```shell
git clone https://github.com/microsoft/DeepSpeedExamples.git
cd DeepSpeedExamples
git checkout efacebb
```

The script train_bert_ds.py is located in the DeepSpeedExamples/HelloDeepSpeed/ directory.
Since the script is not launched using the deepspeed launcher, it needs to read the local_rank from the environment.
The following content has been added at line 670:
```
local_rank = int(os.getenv('LOCAL_RANK', '-1'))
```

### Build Image

The default image name and tag is `kubeflow/pytorch-deepspeed-demo:latest`.

```shell
docker build -f Dockerfile -t kubeflow/pytorch-deepspeed-demo:latest ./
```

### Create the PyTorchJob with DeepSpeed example

```shell
kubectl create -f pytorch_deepspeed_demo.yaml
```
