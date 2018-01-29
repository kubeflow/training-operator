
## Directories Layout

```sh
$ tree -d -I 'vendor|bin|.git'
.
├── build
│   ├── images
│   │   └── tf_operator
│   └── release
├── cmd
│   └── tf_operator
│       └── app
│           └── options
├── dashboard
│   ├── backend
│   │   ├── client
│   │   └── handler
│   └── frontend
│       ├── public
│       └── src
│           └── components
├── docs
│   └── diagrams
├── examples
│   ├── charts
│   │   └── tensorboard
│   │       └── templates
│   ├── crd
│   ├── gke
│   │   └── notebook_image
│   ├── tensorflow-models
│   ├── tf_job
│   │   └── templates
│   └── tf_sample
│       └── tf_sample
├── hack
│   ├── grpc_tensorflow_server
│   └── scripts
├── pkg
│   ├── apis
│   │   └── tensorflow
│   │       ├── helper
│   │       ├── v1alpha1
│   │       └── validation
│   ├── client
│   │   ├── clientset
│   │   │   └── versioned
│   │   │       ├── fake
│   │   │       ├── scheme
│   │   │       └── typed
│   │   │           └── tensorflow
│   │   │               └── v1alpha1
│   │   │                   └── fake
│   │   ├── informers
│   │   │   └── externalversions
│   │   │       ├── internalinterfaces
│   │   │       └── tensorflow
│   │   │           └── v1alpha1
│   │   └── listers
│   │       └── tensorflow
│   │           └── v1alpha1
│   ├── controller
│   ├── trainer
│   └── util
│       ├── k8sutil
│       └── retryutil
├── py
├── test
│   ├── e2e
│   └── test-infra
│       └── airflow
│           └── dags
├── tf-job-operator-chart
│   └── templates
│       └── tests
└── version


```

## Building the Operator

Create a symbolic link inside your GOPATH to the location you checked out the code

```sh
mkdir -p ${GOPATH}/src/github.com/tensorflow
ln -sf ${GIT_TRAINING} ${GOPATH}/src/github.com/tensorflow/k8s
```

* GIT_TRAINING should be the location where you checked out https://github.com/tensorflow/k8s

Resolve dependencies (if you don't have glide install, check how to do it [here](https://github.com/Masterminds/glide/blob/master/README.md#install))

Install dependencies, `-v` will ignore subpackage vendor

```sh
glide install -v
```

Build it

```sh
go install github.com/tensorflow/k8s/cmd/tf_operator
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

We can configure the operator to run locally using the configuration available in your kubeconfig to communicate with
a K8s cluster. Set your environment:

```sh
export KUBECONFIG=$(echo ~/.kube/config)
export MY_POD_NAMESPACE=default
export MY_POD_NAME=my-pod
```

* MY_POD_NAMESPACE is used because the CRD is namespace scoped and we use the namespace of the controller to
  set the corresponding namespace for the resource.
* TODO(jlewi): Do we still need to set MY_POD_NAME? Why?

Now we are ready to run operator locally:

```sh
kubectl create -f examples/crd/crd.yaml
tf_operator --logtostderr
```

The first command creates a CRD `tfjobs`. And the second command runs the operator locally. To verify local
operator is working, create an example job and you should see jobs created by it.

```sh
kubectl create -f https://raw.githubusercontent.com/tensorflow/k8s/master/examples/tf_job.yaml
```

## Go version

On ubuntu the default go package appears to be gccgo-go which has problems see [issue](https://github.com/golang/go/issues/15429) golang-go package is also really old so install from golang tarballs instead.

## Code Style

### Python

* Use two spaces for indents in keeping with Python style
* To autoformat code

  ```sh
  autopep8 -i --indent-size=2 path/to/module.py
  ```

* To sort imports

  ```sh
  isort path/to/module.py
  ```
