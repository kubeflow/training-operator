
# coding: utf-8

# # TF on GKE
# 
# This notebook shows how to run the [TensorFlow CIFAR10 sample](https://github.com/tensorflow/models/tree/master/tutorials/image/cifar10_estimator) on GKE using TfJobs

# ## Requirements

# To run this notebook you must have the following installed
#   * gcloud
#   * kubectl
#   * helm
#   * kubernetes python client library
#   
# There is a Docker image based on Datalab suitable for running this notebook.
# 
# You can start that container as follows
# 
# ```
# docker run --name=gke-datalab -p "127.0.0.1:8081:8080" \
#     -v "${HOME}:/content/datalab/home" \
#     -v /var/run/docker.sock:/var/run/docker.sock -d  -e "PROJECT_ID=" \
#     gcr.io/tf-on-k8s-dogfood/gke-datalab:v20171025-28df43b-dirty
# ```
#   * You need to map in docker so that we can build docker images inside the container.

# ## Preliminaries

# In[1]:


# Turn on autoreloading
get_ipython().run_line_magic('load_ext', 'autoreload')
get_ipython().run_line_magic('autoreload', '2')


# import a bunch of modules and set some constants

# In[1]:


from __future__ import print_function

import os
import sys

ROOT_DIR = os.path.abspath(os.path.join("../.."))
sys.path.append(ROOT_DIR)

import kubernetes
from kubernetes import client as k8s_client
from kubernetes import config as k8s_config
from kubernetes.client.rest import ApiException
import datetime
from googleapiclient import discovery
from googleapiclient import errors
from oauth2client.client import GoogleCredentials
import logging
from pprint import pprint
from py import build_and_push_image
import StringIO
import subprocess
import time
import yaml

logging.getLogger().setLevel(logging.INFO)

TF_JOB_GROUP = "mlkube.io"
TF_JOB_VERSION = "v1beta1"
TF_JOB_PLURAL = "tfjobs"
TF_JOB_KIND = "TfJob"


# ### Configure the notebook for your use
# Change the constants defined below.
#   1. Change **project** to a project you have access to.
#      * GKE should be enabled for that project
#   1. Change **data_dir** and **job_dir**
#      * Use a GCS bucket that you have access to
#      * Ensure the service account on your GKE cluster can read/write to this GCS bucket
# 
# * Optional change the cluster name

# In[2]:


project="cloud-ml-dev"
zone="us-east1-d"
cluster_name="gke-tf-example"
registry = "gcr.io/" + project
data_dir = "gs://cloud-ml-dev_jlewi/cifar10/data"
job_dirs = "gs://cloud-ml-dev_jlewi/cifar10/jobs"
gke = discovery.build("container", "v1")
namespace = "default"


# ### Some Utility Functions

# Execute the cell below to define some utility functions

# In[3]:


def run(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  subprocess.check_call(command, cwd=cwd)

class TimeoutError(Exception):
  """An error indicating an operation timed out."""

def wait_for_operation(client,
                       project,
                       zone,
                       op_id,
                       timeout=datetime.timedelta(hours=1),
                       polling_interval=datetime.timedelta(seconds=5)):
  """Wait for the specified operation to complete.

  Args:
    client: Client for the API that owns the operation.
    project: project
    zone: Zone. Set to none if its a global operation
    op_id: Operation id.
    timeout: A datetime.timedelta expressing the amount of time to wait before
      giving up.
    polling_interval: A datetime.timedelta to represent the amount of time to
      wait between requests polling for the operation status.

  Returns:
    op: The final operation.

  Raises:
    TimeoutError: if we timeout waiting for the operation to complete.
  """
  endtime = datetime.datetime.now() + timeout
  while True:
    if zone:
      op = client.projects().zones().operations().get(
          projectId=project, zone=zone,
          operationId=op_id).execute()
    else:
      op = client.globalOperations().get(project=project,
                                         operation=op_id).execute()

    status = op.get("status", "")
    # Need to handle other status's
    if status == "DONE":
      return op
    if datetime.datetime.now() > endtime:
      raise TimeoutError("Timed out waiting for op: {0} to complete.".format(
          op_id))
    time.sleep(polling_interval.total_seconds())


def configure_kubectl():
  logging.info("Configuring kubectl")
  run(["gcloud", "--project=" + project, "container",
       "clusters", "--zone=" + zone, "get-credentials", cluster_name])    

def create_cluster(gke, name, project, zone):
  """Create the cluster.

  Args:
    gke: Client for GKE.

  """
  cluster_request = {
      "cluster": {
          "name": name,
          "description": "A GKE cluster for TF.",
          "initialNodeCount": 1,
          "nodeConfig": {
              "machineType": "n1-standard-8",
              "oauthScopes": [
                "https://www.googleapis.com/auth/cloud-platform",
              ],
          },
      }
  }
  request = gke.projects().zones().clusters().create(body=cluster_request,
                                                     projectId=project,
                                                     zone=zone)

  try:
    logging.info("Creating cluster; project=%s, zone=%s, name=%s", project,
                 zone, name)
    response = request.execute()
    logging.info("Response %s", response)
    create_op = wait_for_operation(gke, project, zone, response["name"])
    logging.info("Cluster creation done.\n %s", create_op)

  except errors.HttpError as e:
    logging.error("Exception occured creating cluster: %s, status: %s",
                  e, e.resp["status"])
    # Status appears to be a string.
    if e.resp["status"] == '409':      
      pass
    else:
      raise  


# ## GKE Cluster Setup

# * The instructions below create a **CPU** cluster
# * To create a GKE cluster with GPUs sign up for the [GKE GPU Alpha](https://goo.gl/forms/ef7eh2x00hV3hahx1)
# * TODO(jlewi): Update code once GPUs are in beta.
# 
# To use an existing GKE cluster call **configure_kubectl** but not **create_cluster**

# In[106]:


create_cluster(gke, cluster_name, project, zone)      

configure_kubectl()


# ### Install the Operator

# In[107]:


run(["helm", "init"])


# In[ ]:


CHART="https://storage.googleapis.com/tf-on-k8s-dogfood-releases/latest/tf-job-operator-chart-latest.tgz"
run(["helm", "install", CHART, "-n", "tf-job", "--wait", "--replace"])


# ## Build Docker images

# We need to build separate Docker images for CPU and GPU versions of TensorFlow.
#   * **modes** controls whether we build images for CPU, GPU or both 
#   
# The base images controls which version of TensorFlow we will use
#   * Change the base images if you want to use a different version.
#   

# In[5]:


reload(build_and_push_image)

#modes = ["cpu"]
modes = ["cpu", "gpu"]

image = os.path.join(registry, "tf-models")
dockerfile = os.path.join(ROOT_DIR, "examples", "tensorflow-models", "Dockerfile.template")
base_images = {
  "cpu": "gcr.io/tensorflow/tensorflow:1.3.0",
  "gpu": "gcr.io/tensorflow/tensorflow:1.3.0-gpu",
}
images = build_and_push_image.build_and_push(dockerfile, image, modes=modes, base_images=base_images)


# ## Create the CIFAR10 Datasets
# 
# We need to create the cifar10 TFRecord files by running [generate_cifar10_tfrecords.py](https://github.com/tensorflow/models/blob/master/tutorials/image/cifar10_estimator/generate_cifar10_tfrecords.py)
#   * We submit a K8s job to run this program
#   * You can skip this step if your data is already available in data_dir

# In[86]:


k8s_config.load_kube_config()
api_client = k8s_client.ApiClient()
batch_api = k8s_client.BatchV1Api(api_client)

job_name = "cifar10-data-"+ datetime.datetime.now().strftime("%y%m%d-%H%M%S")

body = {}
body['apiVersion'] = "batch/v1"
body['kind'] = "Job"
body['metadata'] = {}
body['metadata']['name'] = job_name
body['metadata']['namespace'] = namespace

# Note backoffLimit requires K8s >= 1.8
spec = """
backoffLimit: 4
template:
  spec:
    containers:
    - name: cifar10
      image: {image}
      command: ["python",  "/tensorflow_models/tutorials/image/cifar10_estimator/generate_cifar10_tfrecords.py", "--data-dir={data_dir}"]
    restartPolicy: Never
""".format(data_dir=data_dir, image=images["cpu"])

spec_buffer = StringIO.StringIO(spec)
body['spec'] = yaml.load(spec_buffer)

try: 
    # Create a Resource
    api_response = batch_api.create_namespaced_job(namespace, body)
    print("Created job %s" % api_response.metadata.name)
except ApiException as e:
    print(
        "Exception when calling DefaultApi->apis_fqdn_v1_namespaces_namespace_resource_post: %s\n" % 
        e)


# wait for the job to finish

# In[87]:


while True:
  results = batch_api.read_namespaced_job(job_name, namespace)
  if results.status.succeeded >= 1 or results.status.failed >= 3:
    break
  print("Waiting for job %s ...." % results.metadata.name)
  time.sleep(5)

if results.status.succeeded >= 1:
  print("Job completed successfully")
else:
  print("Job failed")


# ## Create a TfJob

# To submit a TfJob, we define a TfJob spec and then create it in our cluster

# In[88]:


k8s_config.load_kube_config()
api_client = k8s_client.ApiClient()
crd_api = k8s_client.CustomObjectsApi(api_client)

namespace = "default"
job_name = "cifar10-"+ datetime.datetime.now().strftime("%y%m%d-%H%M%S")
job_dir = os.path.join(job_dirs, job_name)
num_steps = 10
body = {}
body['apiVersion'] = TF_JOB_GROUP + "/" + TF_JOB_VERSION
body['kind'] = TF_JOB_KIND
body['metadata'] = {}
body['metadata']['name'] = job_name
body['metadata']['namespace'] = namespace

spec = """
  replicaSpecs:
    - replicas: 1
      tfReplicaType: MASTER
      template:
        spec:
          containers:
            - image: {image}
              name: tensorflow
              command:
                - python
                - /tensorflow_models/tutorials/image/cifar10_estimator/cifar10_main.py
                - --data-dir={data_dir}
                - --job-dir={job_dir}
                - --train-steps={num_steps}
                - --num-gpus=0
          restartPolicy: OnFailure
  tfImage: {image}
  tensorBoard:
    logDir: {job_dir}
""".format(image=images["cpu"], data_dir=data_dir, job_dir=job_dir, num_steps=num_steps)

spec_buffer = StringIO.StringIO(spec)
body['spec'] = yaml.load(spec_buffer)

try: 
    # Create a Resource
    api_response = crd_api.create_namespaced_custom_object(TF_JOB_GROUP, TF_JOB_VERSION, namespace, TF_JOB_PLURAL, body) 
    logging.info("Created job %s", api_response["metadata"]["name"])
except ApiException as e:
    print(
        "Exception when calling DefaultApi->apis_fqdn_v1_namespaces_namespace_resource_post: %s\n" % 
        e)


# ## Monitoring your job and waiting for it to finish

# In[96]:


configure_kubectl()


# We can monitor the job a number of ways
#   * We can poll K8s to get the status of the TfJob
#   * We can check the TensorFlow logs
#       * These are available in StackDriver
#   * We can access TensorBoard if the TfJob was configured to launch TensorBoard
#   
# Running the code below will poll K8s for the TfJob status and also print out relevant links to monitor the job

# In[98]:


from kubernetes.client.models.v1_label_selector import V1LabelSelector
import urllib2
# Get pod logs
k8s_config.load_kube_config()
api_client = k8s_client.ApiClient()
v1 = k8s_client.CoreV1Api(api_client)

k8s_config.load_kube_config()
api_client = k8s_client.ApiClient()
crd_api = k8s_client.CustomObjectsApi(api_client)

master_started = False
runtime_id = None
while True:
  results = crd_api.get_namespaced_custom_object(TF_JOB_GROUP, TF_JOB_VERSION, namespace, TF_JOB_PLURAL, job_name)

  if not runtime_id:
    runtime_id = results["spec"]["RuntimeId"]
    logging.info("Job has runtime id: %s", runtime_id)
    
    tensorboard_url = "http://127.0.0.1:8001/api/v1/proxy/namespaces/{namespace}/services/tensorboard-{runtime_id}:80/".format(
    namespace=namespace, runtime_id=runtime_id)
    logging.info("Tensorboard will be available at job\n %s", tensorboard_url)

  if not master_started:
    # Get the master pod
    # TODO(jlewi): V1LabelSelector doesn't seem to help
    pods = v1.list_namespaced_pod(namespace=namespace, label_selector="runtime_id={0},job_type=MASTER".format(runtime_id))

    # TODO(jlewi): We should probably handle the case where more than 1 pod gets started.
    # TODO(jlewi): Once GKE logs pod labels we can just filter by labels to get all logs for a particular task
    # and not have to identify the actual pod.
    if pods.items:
      pod = pods.items[0]

      logging.info("master pod is %s", pod.metadata.name)
      query={
        'advancedFilter': 'resource.type="container"\nresource.labels.namespace_id="default"\nresource.labels.pod_id="{0}"'.format(pod.metadata.name), 
        'dateRangeStart': pod.metadata.creation_timestamp.isoformat(),
        'expandAll': 'false',
        'interval': 'NO_LIMIT',
        'logName': 'projects/{0}/logs/tensorflow'.format(project),
       'project': project, 
      }
      logging.info("Logs will be available in stackdriver at\n"
                   "https://console.cloud.google.com/logs/viewer?" + urllib.urlencode(query))
      master_started = True

  if results["status"]["phase"] == "Done":
    break
  print("Job status {0}".format(reslults["status"]["phase"]))
  time.sleep(5)
  
logging.info("Job %s", results["status"]["state"])


# ## Appendix

# In[24]:


from kubernetes.client.models.v1_label_selector import V1LabelSelector
import urllib2
# Get pod logs
k8s_config.load_kube_config()
api_client = k8s_client.ApiClient()
v1 = k8s_client.CoreV1Api(api_client)
runtime_id = results["spec"]["RuntimeId"]
# TODO(jlewi): V1LabelSelector doesn't seem to help
pods = v1.list_namespaced_pod(namespace=namespace, label_selector="runtime_id={0},job_type=MASTER".format(runtime_id))

pod = pods.items[0]


# #### Read the Pod Logs From K8s

# We can read pod logs directly from K8s and not depend on stackdriver

# In[30]:


ret = v1.read_namespaced_pod_log(namespace=namespace, name=pod.metadata.name)
print(ret)


# #### Fetch Logs from StackDriver Programmatically
#   * On GKE pod logs are stored in stackdriver
#   * These logs will stick around longer than pod logs
#   * Fetching from stackNote this tends to be a little slow()

# In[26]:


from google.cloud import logging as gcp_logging
pod_filter = 'resource.type="container" AND resource.labels.pod_id="master-hrhh-0-wrh6g"'
client = gcp_logging.Client(project=project)

for entry in client.list_entries(filter_=pod_filter):
  print(entry.payload.strip())

