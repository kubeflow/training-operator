# MonitoredTrainingSession

## Create TFJob YAML

```
kubectl create -f ./tf_job_mnist.yaml
```

## Build the Image

Download MNIST dataset from [yann.lecun.com/exdb/mnist/](http://yann.lecun.com/exdb/mnist/) and place it in data directory. 

```bash
$ tree data
data
├── t10k-images-idx3-ubyte.gz
├── t10k-labels-idx1-ubyte.gz
├── train-images-idx3-ubyte.gz
└── train-labels-idx1-ubyte.gz

0 directories, 4 files
```

Then you can run:

```
docker build -f Dockerfile -t ghcr.io/kubeflow-incubator/monitored_training_session:1.0 .
```
