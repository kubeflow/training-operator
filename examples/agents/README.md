# Tensorflow/agents

Running a [tensorflow/agents](https://github.com/tensorflow/agents) job on kubernetes using the tf/k8s CRD.

A TfJob YAML can be configured and run using jinja2 and kubectl:

```bash
LOG_DIR=<gcs-bucket-path>
# e.g. gs://${PROJECT_ID}-k8s/logs/tf-v2017111800001
AGENTS_CPU=gcr.io/dev01-181118-181500/agents-example
jinja2 deployment.yaml.template \
   -D image=${AGENTS_CPU} \
   -D job_name=tfagents \
   -D log_dir=${LOG_DIR} \
   -D environment=pybullet_ant \
   | kubectl create -f -
```

This runs a job using the tensorflow/agents example container. To deploy and train custom models you'll need to build and deploy your own container such as via the following:

```bash
gcloud container builds submit \
  --tag gcr.io/<gcloud-project-id>/agents:cpu .
```
