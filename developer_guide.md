# Developer Guide

There are two versions of operator: one for v1alpha1 and one for v1alpha2.

## Building the operator

Create a symbolic link inside your GOPATH to the location you checked out the code

```sh
mkdir -p ${GOPATH}/src/github.com/kubeflow
ln -sf ${GIT_TRAINING} ${GOPATH}/src/github.com/kubeflow/tf-operator
```

* GIT_TRAINING should be the location where you checked out https://github.com/kubeflow/tf-operator

Resolve dependencies (if you don't have dep install, check how to do it [here](https://github.com/golang/dep))

Install dependencies

```sh
dep ensure
```

Build it

```sh
go install github.com/kubeflow/tf-operator/cmd/tf-operator
```

If you want to build the operator for v1alpha2, please use the command here:

```sh
go install github.com/kubeflow/tf-operator/cmd/tf-operator.v2
```

## Building all the artifacts.

[pipenv](https://docs.pipenv.org/) is recommended to manage local Python environment.
You can find setup information on their website.

To build the following artifacts:

* Docker image for the operator
* Helm chart for deploying it

You can run

```sh
# to setup pipenv you have to step into the directory where Pipfile is located
cd py
pipenv install
pipenv shell
cd ..
python -m py.release local --registry=${REGISTRY}
```

* The docker image will be tagged into your registry
* The helm chart will be created in **./bin**

## Running the Operator Locally

Running the operator locally (as opposed to deploying it on a K8s cluster) is convenient for debugging/development.

### Run a Kubernetes cluster

First, you need to run a Kubernetes cluster locally. There are lots of choices:

- [local-up-cluster.sh in Kubernetes](https://github.com/kubernetes/kubernetes/blob/master/hack/local-up-cluster.sh)
- [minikube](https://github.com/kubernetes/minikube)

`local-up-cluster.sh` runs a single-node Kubernetes cluster locally, but Minikube runs a single-node Kubernetes cluster inside a VM. It is all compilable with the controller, but the Kubernetes version should be `1.8` or above.

Notice: If you use `local-up-cluster.sh`, please make sure that the kube-dns is up, see [kubernetes/kubernetes#47739](https://github.com/kubernetes/kubernetes/issues/47739) for more details.

### Configure KUBECONFIG and KUBEFLOW_NAMESPACE

We can configure the operator to run locally using the configuration available in your kubeconfig to communicate with
a K8s cluster. Set your environment:

```sh
export KUBECONFIG=$(echo ~/.kube/config)
export KUBEFLOW_NAMESPACE=$(your_namespace)
```

* KUBEFLOW_NAMESPACE is used when deployed on Kubernetes, we use this variable to create other resources (e.g. the resource lock) internal in the same namespace. It is optional, use `default` namespace if not set.

### Create the TFJob CRD

After the cluster is up, the TFJob CRD should be created on the cluster.

```bash
# If you are using v1alpha1
kubectl create -f ./examples/crd/crd.yml
```

Or

```bash
# If you are using v1alpha2
kubectl create -f ./examples/crd/crd-v1alpha2.yaml
```

### Run Operator

Now we are ready to run operator locally:

```sh
tf-operator
```

To verify local operator is working, create an example job and you should see jobs created by it.

```sh
# If you are using v1alpha1
kubectl create -f ./examples/tf_job.yaml
```

Or

```bash
# If you are using v1alpha2
cd ./test/e2e/dist-mnist
docker build -f Dockerfile -t kubeflow/tf-dist-mnist-test:1.0 .
kubectl create -f ./tf-job-mnist.yaml
```

## Go version

On ubuntu the default go package appears to be gccgo-go which has problems see [issue](https://github.com/golang/go/issues/15429) golang-go package is also really old so install from golang tarballs instead.

## Code Style

### Python

* Use [yapf](https://github.com/google/yapf) to format Python code
* `yapf` style is configured in `.style.yapf` file
* To autoformat code

  ```sh
  yapf -i py/**/*.py
  ```

* To sort imports

  ```sh
  isort path/to/module.py
  ```
