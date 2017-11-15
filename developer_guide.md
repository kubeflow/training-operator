
## Building the Operator

Create a symbolic link inside your GOPATH to the location you checked out the code

```sh
mkdir -p ${GOPATH}/src/github.com/tensorflow
ln -sf ${GIT_TRAINING} ${GOPATH}/src/github.com/tensorflow/k8s
```

  * GIT_TRAINING should be the location where you checked out https://github.com/tensorflow/k8s

Resolve dependencies (if you don't have glide install, check how to do it [here](https://github.com/Masterminds/glide/blob/master/README.md#install))

```sh
glide install
rm -rf  vendor/k8s.io/apiextensions-apiserver/vendor
```

  * The **rm** is needed to remove the vendor directory of dependencies
    that also vendor dependencies as these produce conflicts
    with the versions vendored by mlkube

Build it

```sh
go install github.com/tensorflow/k8s/cmd/tf_operator
```

## Building all the artifacts.

To build the following artifacts:

  * Docker image for the operator
  * Helm chart for deploying it

You can run

```sh
python -m release.py --registry=${REGISTRY}
```
  * The docker image will be tagged into your registry
  * The helm chart will be created in **./bin**


## Running the Operator Locally

Running the operator locally (as opposed to deploying it on a K8s cluster) is convenient for debugging/development.

We can configure the operator to run locally using the configuration available in your kubeconfig to communicate with
a K8s cluster.

Set your environment
```sh
export KUBECONFIG=$(echo ~/.kube/config)
export MY_POD_NAMESPACE=default
export MY_POD_NAME=my-pod
```

    * MY_POD_NAMESPACE is used because the CRD is namespace scoped and we use the namespace of the controller to
      set the corresponding namespace for the resource.

TODO(jlewi): Do we still need to set MY_POD_NAME? Why?

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
