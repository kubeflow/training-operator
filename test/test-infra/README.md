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


## Accessing the Airflow UI

The best way to connect to the Airflow UI is using kubectl to forward a port

```
kubectl port-forward `kubectl get pods --selector=name=airflow -o jsonpath='{.items[*].metadata.name}'` 8180:8080
```

  * Using the ApiServer proxy doesn't work so well because redirects won't work.
  * You will need to have access to the K8s cluster where we run Airflow.

## Running the E2E tests locally

Each step in our Airflow pipeline just invokes a binary. So any of the steps can be performed locally just
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