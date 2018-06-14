### Distributed mnist model for e2e test

This folder containers Dockerfile and distributed mnist model for e2e test.

**Build Image**

The default image name and tag is `kubeflow/tf-dist-mnist-test:1.0`.

```shell
docker build -f Dockerfile -t kubeflow/tf-dist-mnist-test:1.0 ./
```

**Create TFJob YAML**

```
kubectl create -f ./tf_job_mnist.yaml
```
