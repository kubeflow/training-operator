# Test Infrastructure

Please refer to the [testing repository](https://github.com/kubeflow/testing) for general information
that applies to all Kubeflow repositories.

Quick Links
 * [config.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/config.yaml)
defines the ProwJobs.
  * Search for tf-k8s to find tf-k8s related jobs
 * [tf-k8s Test Results Dashboard](https://k8s-testgrid.appspot.com/sig-big-data)
 * [tf-k8s Prow Jobs dashboard](https://prow.k8s.io/?repo=tensorflow%2Fk8s)

## Anatomy of our Tests

Our tests are defined as an Argo workflow defined using ksonnet.

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
