# mlkube Test Infrastructure

We use [Prow](https://github.com/kubernetes/test-infra/tree/master/prow)
K8s continuous integration tool.

Prow is a set of binaries that run on Kubernetes and respond to
GitHub events.

[config.yaml](config.yaml) defines the ProwJobs we want to run.

Our ProwJobs use the Docker image defined in [image](image)

## Testing Changes to the ProwJobs

Follow Prow's
[getting started guide](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md)
to create your own prow cluster.

Checkout [kubernetes/test-infra](https://github.com/kubernetes/test-infra).

```
git clone https://github.com/kubernetes/test-infra git_k8s-test-infra
```

Build the mkpj binary

```
bazel build build prow/cmd/mkpj
```

Generate the ProwJob Config

```
./bazel-bin/prow/cmd/mkpj/mkpj --job=$JOB_NAME --config-path=$CONFIG_PATH
```
    * This binary will prompt for needed information like the sha #
    * The output will be a ProwJob spec which can be instantiated using
       kubectl
