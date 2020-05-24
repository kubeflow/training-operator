# Multi-worker training with Keras

This directory contains a example for running multi-worker distributed training 
using Tensorflow 2.1 keras API on Kubeflow. For more information about the 
source code, please see TensorFlow tutorials [here](https://www.tensorflow.org/tutorials/distribute/keras) and [here](https://www.tensorflow.org/tutorials/distribute/multi_worker_with_keras)

## Prerequisite

Your cluster must be configured to use Multiple GPUs, 
please follow the [instructions](https://www.kubeflow.org/docs/components/training/tftraining/#using-gpus)

## Steps

1.  Build a image
    ```
    docker build -f Dockerfile -t kubeflow/multi_worker_strategy:v1.0 .
    ```

2.  Specify your storageClassName and create a persistent volume claim to save 
    models and checkpoints
    ```
    kubectl -n ${NAMESPACE} create -f pvc.yaml
    ```

3.  Create a TFJob, if you use some GPUs other than NVIDIA, please replace 
    `nvidia.com/gpu` with your GPU vendor in the `limits` section.
    ```
    kubectl -n ${NAMESPACE} create -f multi_worker_tfjob.yaml
    ```
