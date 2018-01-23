# Airflow for writing E2E pipelines.

We are currently experimenting with using Airflow as a way to express our E2E
pipelines. For background see [issues#120](https://github.com/tensorflow/k8s/issues/120).

To facilitate developing and debugging our pipelines we use the following conventions
  * Each step in the Airflow pipeline should be a binary
     * This makes it easy to reproduce/debug a step just by invoking the binary
       with the required parameters.
     * For convenience, we often structure our binaries e.g. deploy.py and release.py as binaries with multiple commands where each command corresponds to a different step in the Airflow.

  * We mostly use the PythonOperator step and then invoke each step as a subprocess
    * Each step supports a dryrun mode in which we print out the subprocess command but don't execute it
      * This facilitates debugging the pipeline especially XCom issues.
  	* This allows us to process run config using Python
  * We rely on Airflow's xcom features to communicate data between steps in the pipeline.

## Deploying Airflow

* Currently we bake the DAGs into the container [image](Dockerfile)
* We deploy Airflow on K8s using the LocalExecutor and a postgre database
   * [deployment.yaml](deployment.yaml)
   * Eventually we'll switch to the [Airflow K8s executor](https://cwiki.apache.org/confluence/pages/viewpage.action?pageId=71013666)
* Its currently running on the GKE cluster **prow** in project **mlkube-testing**

### One time setup

Create a PD to store the POSTGRES data.

```
gcloud --project=${PROJECT} compute disks create --size=200GB --zone=${ZONE} airflow-data
```

Create a service account to be used by Airflow to access GCP services

```
NODE_SERVICE_ACCOUNT=${PROJECT_NUMBER}-compute@developer.gserviceaccount.com
gcloud iam service-accounts --project=${PROJECT} create --display-name="Airflow" ${SERVICE_ACCOUNT}
gcloud iam service-accounts keys --project=${PROJECT} create --iam-account=${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com ~/${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com.key.json
gcloud projects add-iam-policy-binding ${PROJECT} --member serviceAccount:${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com  --role roles/storage.admin
gcloud projects add-iam-policy-binding ${PROJECT} --member serviceAccount:${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com  --role roles/cloudbuild.builds.editor
gcloud projects add-iam-policy-binding ${PROJECT} --member serviceAccount:${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com  --role roles/logging.viewer
gcloud projects add-iam-policy-binding ${PROJECT} --member serviceAccount:${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com  --role roles/viewer
gcloud projects add-iam-policy-binding ${PROJECT} --member serviceAccount:${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com  --role roles/container.admin
gcloud projects add-iam-policy-binding ${PROJECT} --member serviceAccount:${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com  --role roles/container.admin
gcloud iam service-accounts --project=${PROJECT} add-iam-policy-binding --role=roles/iam.serviceAccountUser --member=serviceAccount:${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com  ${NODE_SERVICE_ACCOUNT}
kubectl create secret generic airflow-key --from-file=key.json=~/${SERVICE_ACCOUNT}@${PROJECT}.iam.gserviceaccount.com.key.json
```
  * The service account used by our Airflow pipeline needs ServiceAccountActor permission on the service account used with nodes in the GKE cluster in order to 
    create a GKE cluster.
  * Due to https://github.com/GoogleCloudPlatform/cloud-builders/issues/120 service account needs to be a viewer in order for gcloud container builds
    to work.

### Updating the DAGs

The DAGs are currently baked into the Airflow Docker container. To update the DAGS

```
make push
kubectl apply -f deployment.yaml
```

### Accessing the UI

You can access the UI over kubectl proxy

```
http://${PROXY}/api/v1/proxy/namespaces/default/services/airflow:80/admin/
```
  * TODO(jlewi): This doesn't work so well because Airflow redirects won't work as expected. Using NodePort and then creating an SSH
    tunnel might work better.
      * Oftentimes you can make a given URL work (e.g. the URL for logs) by copying the link and appending it to the URL given
        above.


## Running Airflow locally

Here are some instructions for running Airflow locally

* The makefile contains commands to start Airflow and POSTGRE locally using docker

### GCP Credentials

Some of the steps in our Airflow pipeline require GCP credentials. The easiest way to handle credentials
is to create a service account with an associated private key. You can then volume mount the credentials
into the Docker container. 

Follow the commands [above](#one-time-setup) to create a service account.

Alternatively if don't want to use a service account you can run the following commands inside the container to use
your credentials.


Login 
```
gcloud auth login
gcloud auth application-default login
```
  * The reason we need to issue two login commands is because some code uses the application default credentials but other code just shells out to gcloud so we need to set a default account.

### Start the Airflow and POSTGRE containers

```
make run_postgre
export GOOGLE_APPLICATION_CREDENTIALS=${path/to/your/key}
make run_airflow
```

* The dags are volume mounted from the host machine so that you can pick up changes without restarting
	the airflow container.
* You will probably need to make your DAGs world readable so they are accessible inside the container
* The service account key will be volume mounted into the container.

```
  chmod -R a+rwx ${GIT_TRAINING}/test-infra/airflow/dags
```

* **Only the dags** are mounted from the host; if you make changes to the code invoked by the dags you will need to restart the Airflow container
  * TODO(jlewi): Can we mount code in **py/...** into the container as well? I think this is an issue with permissions. What if we configure Airflow
  account inside the container to use the same userid as the user?

### Accessing Airflow

You can access the Airflow UI at [localhost:8080](http://localhost:8080)

Start a shell inside the Airflow container.
```
docker exec -ti airflow /bin/bash
```

To trigger a DAG run you can do	

```
airflow trigger_dag tf_k8s_tests -conf '{}'
```
 * The value of conf should be serialized JSON representing **dag_run.conf**.

To trigger a dryrun
```
airflow trigger_dag tf_k8s_tests --conf '{"dryrun": true}'
```
