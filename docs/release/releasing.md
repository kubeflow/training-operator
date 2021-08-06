# Releasing the TFJob operator

Permissions

	* You need to be a member of release-team@kubeflow.org to have access to the GCP
	  resources used for releasing.

	* You need write permissions on the repository to create a release branch.


Look at the [postsubmit dashboard](https://k8s-testgrid.appspot.com/sig-big-data#kubeflow-tf-operator-postsubmit)
to find the latest green postsubmit.


Use the GitHub UI to cut a release branch
	* Name the release branch v{MAJOR}.${MINOR}-branch

Checkout the release branch

We build TFJob operator by running the E2E test workflow.

Look at the [postsubmit dashboard](https://k8s-testgrid.appspot.com/sig-big-data#kubeflow-tf-operator-postsubmit)
to find the latest green postsubmit.

Check out that commit (in this example, we'll use `6214e560`):

Run the E2E test workflow using our release cluster

[kubeflow/testing#42](https://github.com/kubeflow/testing/issues/42) will simplify this.

```
submit_release_job.sh ${COMMIT}
```

You can monitor the workflow using the Argo UI. For our release cluster, we don't expose the Argo UI publicly, so you'll need to connect via kubectl port-forward:

```
kubectl -n kubeflow-releasing port-forward `kubectl -n kubeflow-releasing get pods --selector=app=argo-ui -o jsonpath='{.items[0].metadata.name}'` 8080:8001
```

[kubeflow/testing#43](https://github.com/kubeflow/testing/issues/43) is tracking setup of IAP to make this easier.

Make sure the Argo workflow completes successfully.
Check the junit files to make sure there were no actual test failures.
The junit files will be in [gs://kubeflow-releasing-artifacts](https://console.cloud.google.com/storage/browser/kubeflow-releasing-artifacts/logs/kubeflow_tf-operator/tf-operator-release/?project=kubeflow-releasing).
	* The build artifacts will be in a directory named after the build number

If the tests pass use the GitHub UI to create a release tagged v{MAJOR}-{MINOR}-{PATCH}
	* If its an RC append -RC.N
	* In the notes create a link to the Docker image in GCR
	* For the label use the `sha256` and not the label so it is immutable.

To release new ksonnet configs with the image following [kubeflow/kubeflow/releasing.md](https://github.com/kubeflow/kubeflow/blob/master/releasing.md).
