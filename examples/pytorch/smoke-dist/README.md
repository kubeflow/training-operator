### Distributed send/recv e2e test 

This folder containers Dockerfile and distributed send/recv test.

**Build Image**

The default image name and tag is `kubeflow/pytorch-dist-sendrecv-test:1.0`.

```shell
docker build -f Dockerfile -t kubeflow/pytorch-dist-sendrecv-test:1.0 ./
```

**Create the PyTorch job**

```
kubectl create -f ./pytorch_job_sendrecv.yaml
```
