# Tensorflow/agents

Running a [tensorflow/agents](https://github.com/tensorflow/agents) job on kubernetes using the tf/k8s CRD.

# Training a model

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
   -D mode=train
   | kubectl create -f -
```

This runs a job using the tensorflow/agents example container. To deploy and train custom models you'll need to build and deploy your own container such as via the following:

```bash
gcloud container builds submit \
  --tag gcr.io/<gcloud-project-id>/agents:cpu .
```

Example tensorboard results can be accessed via the following

```bash
tensorboard --logdir gs://dev01-181118-181500-k8s/jobs/tensorflow-20171117102413/20171117T182424-pybullet_ant
```

(TODO: Add screen grabs to readme?)

# Rendering

A render job can be initiated as follows, templating the '--mode render' into

```bash
jinja2 deployment.yaml.template \
   -D image=${AGENTS_CPU} \
   -D job_name=tfagents-render \
   -D log_dir=${LOG_DIR} \
   -D environment=pybullet_ant \
   -D mode=render \
   | kubectl create -f -
```

(TODO: Update with gif of render when finished)
