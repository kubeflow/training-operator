# mlkube Test Infrastructure

We use [Prow](https://github.com/kubernetes/test-infra/tree/master/prow)
K8s continuous integration tool.

Prow is a set of binaries that run on Kubernetes and respond to
GitHub events.

[config.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)
defines the ProwJobs for a cluster.

    * Search for mlkube to find mlkube related jobs

Our ProwJobs use the Docker image defined in [image](image)

## Testing Changes to the ProwJobs

Follow Prow's
[getting started guide](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md)
to create your own prow cluster.

    * TODO(jlewi): We don't really need the ingress. You can connect
      over kubectl or some other mechanism.

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

Create the ProwJob

```
kubectl create -f ${PROW_JOB_YAML_FILE}
```

    * To rerun the job bump metadata.name and status.startTime

To monitor the job open Prow's UI by navigating to the exteral IP
associated with the ingress for your Prow cluster.

## Integration with K8s Prow Infrastructure.

We rely on K8s instance of Prow to actually run our jobs.

Here's [a dashboard](https://k8s-testgrid.appspot.com/sig-big-data) with
the results.

Our jobs should be added to
[K8s config](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)

## Running the tests locally.

* The ProwJobs invoke [runner.py](image/runner.py) inside this [container.](image/Dockerfile)
* You can invoke runner.py to run the E2E tests on your local Changes
    * You will need to set the arguments to use a GKE cluster in a project
      you have access too.
* You can also run it inside the same Docker image,
  gcr.io/mlkube-testing/builder:latest, used by prow

  ```
  docker run -ti -v ${TRAINING_PATH}:/go/src/github.com/jlewi/mlkube.io \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --entrypoint=/bin/bash gcr.io/mlkube-testing/builder:latest
  gcloud auth login
  gcloud auth application-default login
  python runner.py --no-gcb
  ```