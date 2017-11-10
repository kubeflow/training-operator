# Test Infrastructure

We use [Prow](https://github.com/kubernetes/test-infra/tree/master/prow),
K8s' continuous integration tool.

  * Prow is a set of binaries that run on Kubernetes and respond to
GitHub events.

We use Prow to run:

  * Presubmit jobs
  * Postsubmit jobs
  * Periodic tests

Quick Links
 * [config.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)
defines the ProwJobs.
  * Search for tf-k8s to find tf-k8s related jobs
 * [tf-k8s Test Results Dashboard](https://k8s-testgrid.appspot.com/sig-big-data)
 * [tf-k8s Prow Jobs dashboard](https://prow.k8s.io/?repo=tensorflow%2Fk8s)

## Anatomy of our Prow Jobs

* Our prow jobs are defined in [config.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)
* Each prow job defines a K8s PodSpec indicating
* Our prow jobs use [bootstrap.py](image/bootstrap.py) to checkout at
  the repository at the commit being tested.
*  [bootstrap.py](image/bootstrap.py) will then invoke
   [runner.py](runner.py)
* [runner.py](runner.py) runs the test
    * Builds and pushes a Docker image for the CRD
    * Creates a GKE cluster
    * Invokes [helm-test/main.go](helm-test/main.go) to deploy the
      helm package and run the helm tests
    * Delete the GKE cluster
    * Copy test results/artifacts to GCS for gubernator
    * TODO(jlewi): runner.py and main.go should be merged into a single
      go program.

## Running the E2E tests manually.

[runner.py](runner.py) can be invoked directly without using Prow
to easily test your local changes.

```
python ./test-infra/runner.py --project=${PROJECT} --zone=${ZONE}
  --src_dir=${REPO_PATH}
```

* Specify a project and zone where the GKE cluster will be created.
* runner.py will perform all steps; e.g.
   * Building Docker images
   * Starting a cluster
   * etc..

If you don't want to use a GKE cluster

```
python ./test-infra/runner.py --no-gke--src_dir=${SRC_PATH}
```

* In this case the test will use whatever cluster kubectl is configured
   to use.

You can also run the tests inside the Docker image,
  gcr.io/mlkube-testing/builder:latest, used by prow
    * This can be useful for debugging or testing changes

  ```
  docker run -ti -v ${REPO_PATH}:/go/src/github.com/tensorflow/k8s \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --entrypoint=/bin/bash gcr.io/mlkube-testing/builder:latest
  gcloud auth login
  gcloud auth application-default login
  python ${REPO_PATH}/test-infra/runner.py --no-gcb
  ```

## Running the test for a specific image

If want to debug a presubmit you may not want to rebuild the Docker
images. In thise case you can rerun the test by doing

```
go install github.com/tensorflow/k8s/test-infra/helm-test
${GOPATH}/bin/helm-test --image=${DOCKER_IMAGE} --output_dir=/tmp/testOutput
```
    * The docker image used by the presubmit should be availabe from
      the presubmit logs.

## Testing Changes to the ProwJobs

Changes to our ProwJob configs in [config.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)
should be relatively infrequent since most of the code invoked
as part of our tests lives in the repository.

However, in the event we need to make changes here are some instructions
for testing them.

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
associated with the ingress for your Prow cluster or using
kubectl proxy.

## Integration with K8s Prow Infrastructure.

We rely on K8s instance of Prow to actually run our jobs.

Here's [a dashboard](https://k8s-testgrid.appspot.com/sig-big-data) with
the results.

Our jobs should be added to
[K8s config](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)

## Notes adding tensorflow/k8s to K8s Prow Instance

Below is some notes on what it took to integrate with K8s Prow instance.

1. Define ProwJobs see [pull/4951](https://github.com/kubernetes/test-infra/pull/4951)

    * Add prow jobs to [prow/config.yaml](https://github.com/kubernetes/test-infra/pull/4951/files#diff-406185368ba7839d1459d3d51424f104)
    * Add trigger plugin to [prow/plugins.yaml](https://github.com/kubernetes/test-infra/pull/4951/files#diff-ae83e55ccb05896d5229df577d34255d)
    * Add test dashboards to [testgrid/config/config.yaml](https://github.com/kubernetes/test-infra/pull/4951/files#diff-49f154cd90facc43fda49a99885e6d17)
    * Modify [testgrid/jenkins_verify/jenkins_validat.go](https://github.com/kubernetes/test-infra/pull/4951/files#diff-7fb4731a02dd681bbd0daada8dd2f908)
       to allow presubmits for the new repo.
1. For tensorflow/k8s configure webhooks by following these [instructions](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md#add-the-webhook-to-github)
    * Use https://prow.k8s.io/hook as the target
    * Get HMAC token from k8s test team
1. Add the k8s bot account, k8s-ci-robot, as an admin on the repository
    * Admin privileges are needed to update status (but not comment)
    * Someone with access to the bot will need to accept the request.
1. TODO(jlewi): Follow [instructions](https://github.com/kubernetes/test-infra/tree/master/gubernator#adding-a-repository-to-the-pr-dashboard) for adding a repository to the PR
   dashboard.