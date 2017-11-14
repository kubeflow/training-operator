# Airflow for writing E2E pipelines.

We are currently experimenting with using Airflow as a way to express our E2E
pipelines. For background see [issues#120](https://github.com/tensorflow/k8s/issues/120).

To facilitate developing and debugging our pipelines we use the following conventions
  * Each step in the Airflow pipeline should be a binary
     * This makes it easy to reproduce/debug a step just by invoking the binary
       with the required parameters.
     * For convenvenience, we often structure our binaries e.g. deploy.py and release.py as binaries with multiple commands where each command corresponds to a different step in the Airflow.

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

## Running Airflow locally

Here are some instructions for running Airflow locally

* The makefile contains commands to start Airflow and POSTGRE locally using docker

### GCP Credentials

Some of the steps in our Airflow pipeline require GCP credentials. The easiest way to handle credentials
is to create a service account with an associated private key. You can then volume mount the credentials
into the Docker container. 

Here are some commands to create a service account

```
gcloud iam service-accounts --project=${PROJECT} create ${SERVICE_ACCOUNT}
gcloud iam service-accounts keys --project=${PROJECT} create ~/${KEYNAME}.json  --iam-account=${SERVICE-ACCOUNT}@${PROJECT}.iam.gserviceaccount.com
# TODO(jlewi): Need to grant appropriate permissions to the service account.
```

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
	```
	chmod -R a+rwx ${GIT_TRAINING}/test-infra/airflow/dags
	```
	* **Only the dags** are mounted from the host; if you make changes to the code invoked by the dags you will need to restart the Airflow container
		* TODO(jlewi): Can we mount code in **py/...** into the container as well? I think this is an issue with permissions. What if we configure Airflow to run as user root and Airflow inside the container?

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