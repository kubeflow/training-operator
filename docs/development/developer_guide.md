# Developer Guide

Kubeflow Training Operator is currently at v1.

## Requirements

- [Go](https://golang.org/) (1.22 or later)

## Building the operator

Create a symbolic link inside your GOPATH to the location you checked out the code

```sh
mkdir -p $(go env GOPATH)/src/github.com/kubeflow
ln -sf ${GIT_TRAINING} $(go env GOPATH)/src/github.com/kubeflow/training-operator
```

- GIT_TRAINING should be the location where you checked out https://github.com/kubeflow/training-operator

Install dependencies

```sh
go mod tidy
```

Build it

```sh
go install github.com/kubeflow/training-operator/cmd/training-operator.v1
```

## Running the Operator Locally

Running the operator locally (as opposed to deploying it on a K8s cluster) is convenient for debugging/development.

### Run a Kubernetes cluster

First, you need to run a Kubernetes cluster locally. We recommend `kind`.

- [kind](https://kind.sigs.k8s.io)


### Configure KUBECONFIG and KUBEFLOW_NAMESPACE

We can configure the operator to run locally using the configuration available in your kubeconfig to communicate with
a K8s cluster. Set your environment:

```sh
export KUBECONFIG=$(echo ~/.kube/config)
export KUBEFLOW_NAMESPACE=trainingoperator
```

- KUBEFLOW_NAMESPACE is used when deployed on Kubernetes, we use this variable to create other resources (e.g. the resource lock) internal in the same namespace. It is optional, use `default` namespace if not set.

You can create a `kind` cluster by running
```sh
kind create cluster --name $KUBEFLOW_NAMESPACE
```

### Create the TFJob CRD

After the cluster is up, the TFJob CRD should be created on the cluster.

```bash
make install
```

### Build Operator Image
```sh
make docker-build IMG=my-username/training-operator:my-pr-01
```
You can swap `my-username/training-operator:my-pr-01` with whatever you would like.

## Load docker image 
```sh
 kind load docker-image my-username/training-operator:my-pr-01
``` 

## Modify operator image with new one
Update the `newTag` key in `./manifests/overlayes/standalone/kustimization.yaml` with the new image.

Deploy the operator with: 
```sh 
kubectl apply -k ./manifests/overlayes/standalone
```
And now we can submit jobs to the operator.

You should be able to see a pod for your training operator running in your namespace using
```commandline
kubectl get namespace
kubectl get pods -n YOUR_NAMESPACE_FROM_ABOVE
kubectl logs -n YOUR_NAMESPACE_FROM_ABOVE YOUR_PODNAME_FROM_ABOVE
```
Please make sure to replace `YOUR_NAMESPACE_FROM_ABOVE` with the namespace you are using and `YOUR_PODNAME_FROM_ABOVE` with the pod name you see from the `kubectl get pods` command.

### Test running an operator locally 
```sh 
cd ./examples/pytorch/mnist2/
docker build -f Dockerfile.cpu .
kubectl create -f ./sample_pytorchjob.yaml
kubectl describe PyTorchJob
kubectl get pods -n YOUR_NAMESPACE_FROM_ABOVE
kubectl events
```
### Run Operator

Now we are ready to run operator locally:

```sh
make run
```
## Go version

On ubuntu the default go package appears to be gccgo-go which has problems see [issue](https://github.com/golang/go/issues/15429) golang-go package is also really old so install from golang tarballs instead.

## Generate Python SDK

To generate Python SDK for the operator, run:

```
./hack/python-sdk/gen-sdk.sh
```

This command will re-generate the api and model files together with the documentation and model tests.
The following files/folders in `sdk/python` are auto-generated and should not be modified directly:

```
sdk/python/docs
sdk/python/kubeflow/training/models
sdk/python/kubeflow/training/*.py
sdk/python/test/*.py
```

The Training Operator client and public APIs are located here:

```
sdk/python/kubeflow/training/api
```

## Code Style

### Python

- Use [`black`](https://github.com/psf/black) to format Python code

- Run the following to install `black`:

  ```
  pip install black==23.9.1
  ```

- To check your code:

  ```sh
  black --check --exclude '/*kubeflow_org_v1*|__init__.py|api_client.py|configuration.py|exceptions.py|rest.py' sdk/
  ```
