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

## Anatomy of our Tests

![Test Infrastructure](test_infrastructure.png)
* Our prow jobs are defined in [config.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)
* Each prow job defines a K8s PodSpec indicating a command to run
* Our prow jobs use [airflow.py](https://github.com/tensorflow/k8s/blob/master/py/airflow.py) to trigger an Airflow pipeline that checks out our code
  and runs our Tests.
* Our tests are structured as Airflow pipelines so that we can easily perform steps in parallel.
* The Airflow pipeline is defined in [e2e_tests_dag.py](airflow/dags/e2e_tests_dag.py)
    * Builds and pushes a Docker image and helm package for the CRD (see [release.py](https://github.com/tensorflow/k8s/blob/master/py/release.py))
    * Creates a GKE cluster (see [deploy.py](https://github.com/tensorflow/k8s/blob/master/py/deploy.py))
    * Deploys the CRD (see [deploy.py](https://github.com/tensorflow/k8s/blob/master/py/deploy.py))
    * Runs the tests (see [deploy.py](https://github.com/tensorflow/k8s/blob/master/py/deploy.py))
    * Delete the GKE cluster (see [deploy.py](https://github.com/tensorflow/k8s/blob/master/py/deploy.py))
    * Copy test results/artifacts to GCS for gubernator


## Running the E2E tests locally

Each step in our Argo workflow just invokes a binary. So any of the steps can be performed locally just
by running the appropriate binary.

For example to manually debug a presubmit you can do something like the following

Build the artifacts for a presubmit

```
python -m py.release pr --registry ${REGISTRY} --project=${PROJECT} --releases_path=${RELEASES_PATH} --pr=${PR} --commit=${COMMIT}

```

Create a GKE cluster and deploy the artifacts

```
python -m py.deploy setup --project=${PROJECT} --zone=${ZONE} --cluster=${CLUSTER} --chart=${CHART}

```  

Run the tests

```
python -m py.deploy test --project=${PROJECT} --zone=${ZONE} --cluster=${CLUSTER}
```  
  * TODO(jlewi): We should probably add an option to run on whatever cluster kubectl is configured to use.

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

To monitor the job open Prow's UI by navigating to the external IP
associated with the ingress for your Prow cluster or using
kubectl proxy.
