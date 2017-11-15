# Tensorflow/agents on k8s

A simple example demonstrating training of a tensorflow/agents model on kubernetes.


## Build and deploy the image

The following will build and push an image that includes the model to be trained to the Google Container Registry (GCR).

```bash
MODEL_IMAGE=gcr.io/your-project/agents_simple
DOCKERFILE_PATH=<path to examples>/agents_simple/Dockerfile

python build_and_push_image.py --image $MODEL_IMAGE \
                               --dockerfile $DOCKERFILE_PATH
```

You can verify GCR registry is accessible from your kubernetes cluster by launching an interactive pod from the above image:

```bash
kubectl run -i --tty image --image=${MODEL_IMAGE} --restart=Never -- sh
```

## Initiate and monitor training

```bash

python runner.py --model_image=$MODEL_IMAGE \
                 --num_workers=3 \
                 --num_ps=2 \
                 --use_gpu=true
```
