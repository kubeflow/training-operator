### Distributed mnist model for e2e test

This folder containers Dockerfile and distributed mnist model for e2e test.

**Build Image**

The default image name and tag is `kubeflow/tf-dist-mnist-test:1.0`.

To build this image on x86_64:
```shell
docker build -f Dockerfile -t kubeflow/tf-dist-mnist-test:1.0 ./
```
To build this image on ppc64le:
```shell
docker build -f Dockerfile.ppc64le -t kubeflow123/tf-dist-mnist-test:1.0 ./
```

**Create TFJob YAML**

```
kubectl create -f ./tf_job_mnist.yaml
```
  * If on ppc64le, please update tf_job_mnist.yaml to use the image of ppc64le firstly.