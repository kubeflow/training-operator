"""Utilities used by our python scripts for building and releasing."""
from __future__ import print_function

import datetime
import logging
import os
import re
import shutil
import subprocess
import time
import urllib
import yaml

import google.auth
import google.auth.transport
import google.auth.transport.requests

from googleapiclient import errors
from kubernetes import client as k8s_client
from kubernetes.config import kube_config
from kubernetes.client import configuration
from kubernetes.client import rest

# Default name for the repo organization and name.
# This should match the values used in Go imports.
MASTER_REPO_OWNER = "tensorflow"
MASTER_REPO_NAME = "k8s"


def run(command, cwd=None, env=None, use_print=False, dryrun=False):
  """Run a subprocess.

  Any subprocess output is emitted through the logging modules.
  """
  logging.info("Running: %s \ncwd=%s", " ".join(command), cwd)

  if not env:
    env = os.environ

  try:
    if dryrun:
      command_str = ("Dryrun: Command:\n{0}\nCWD:\n{1}\n"
                     "Environment:\n{2}").format(" ".join(command), cwd, env)
      if use_print:
        print(command_str)
      else:
        logging.info(command_str)
      return
    output = subprocess.check_output(command, cwd=cwd, env=env,
                                     stderr=subprocess.STDOUT).decode("utf-8")

    if use_print:
      # With Airflow use print to bypass logging module.
      print("Subprocess output:\n")
      print(output)
    else:
      logging.info("Subprocess output:\n%s", output)
  except subprocess.CalledProcessError as e:
    if use_print:
      # With Airflow use print to bypass logging module.
      print("Subprocess output:\n")
      print(e.output)
    else:
      logging.info("Subprocess output:\n%s", e.output)
    raise

def run_and_output(command, cwd=None, env=None):
  logging.info("Running: %s \ncwd=%s", " ".join(command), cwd)

  if not env:
    env = os.environ
  # The output won't be available until the command completes.
  # So prefer using run if we don't need to return the output.
  try:
    output = subprocess.check_output(command, cwd=cwd, env=env,
                                     stderr=subprocess.STDOUT).decode("utf-8")
    logging.info("Subprocess output:\n%s", output)
  except subprocess.CalledProcessError as e:
    logging.info("Subprocess output:\n%s", e.output)
    raise
  return output


def clone_repo(dest, repo_owner=MASTER_REPO_OWNER, repo_name=MASTER_REPO_NAME,
               sha=None, branches=None):
  """Clone the repo,

  Args:
    dest: This is the root path for the training code.
    repo_owner: The owner for github organization.
    repo_name: The repo name.
    sha: The sha number of the repo.
    branches: (Optional): One or more branches to fetch. Each branch be specified
      as "remote:local". If no sha is provided
      we will checkout the last branch provided. If a sha is provided we
      checkout the provided sha.

  Returns:
    dest: Directory where it was checked out
    sha: The sha of the code.
  """
  # Clone mlkube
  repo = "https://github.com/{0}/{1}.git".format(repo_owner, repo_name)
  logging.info("repo %s", repo)

  # TODO(jlewi): How can we figure out what branch
  run(["git", "clone", repo, dest])

  if branches:
    for b in branches:
      run(["git", "fetch", "origin", b,], cwd=dest)

    if not sha:
      b = branches[-1].split(":", 1)[-1]
      run(["git", "checkout", b,], cwd=dest)

  if sha:
    run(["git", "checkout", sha], cwd=dest)

  # Get the actual git hash.
  # This ensures even for periodic jobs which don't set the sha we know
  # the version of the code tested.
  sha = run_and_output(["git", "rev-parse", "HEAD"], cwd=dest)

  # Install dependencies
  run(["glide", "install"], cwd=dest)

  # We need to remove the vendored dependencies of apiextensions otherwise we
  # get type conflicts.
  shutil.rmtree(os.path.join(dest,
                             "vendor/k8s.io/apiextensions-apiserver/vendor"))

  return dest, sha


def to_gcs_uri(bucket, path):
  """Convert bucket and path to a GCS URI."""
  return "gs://" + os.path.join(bucket, path)


def create_cluster(gke, project, zone, cluster_request):
  """Create the cluster.

  Args:
    gke: Client for GKE.
    project: The project to create the cluster in
    zone: The zone to create the cluster in.
    cluster_rquest: The request for the cluster.
  """
  request = gke.projects().zones().clusters().create(body=cluster_request,
                                                     projectId=project,
                                                     zone=zone)

  try:
    logging.info("Creating cluster; project=%s, zone=%s, name=%s", project,
                 zone, cluster_request["cluster"]["name"])
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

def delete_cluster(gke, name, project, zone):
  """Delete the cluster.

  Args:
    gke: Client for GKE.
    name: Name of the cluster.
    project: Project that owns the cluster.
    zone: Zone where the cluster is running.
  """

  request = gke.projects().zones().clusters().delete(clusterId=name,
                                                     projectId=project,
                                                     zone=zone)

  try:
    response = request.execute()
    logging.info("Response %s", response)
    delete_op = wait_for_operation(gke, project, zone, response["name"])
    logging.info("Cluster deletion done.\n %s", delete_op)

  except errors.HttpError as e:
    logging.error("Exception occured deleting cluster: %s, status: %s",
                  e, e.resp["status"])

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

def configure_kubectl(project, zone, cluster_name):
  logging.info("Configuring kubectl")
  run(["gcloud", "--project=" + project, "container",
       "clusters", "--zone=" + zone, "get-credentials", cluster_name])

def wait_for_tiller_to_be_ready(api_client):
  # Wait for tiller to be ready
  end_time = datetime.datetime.now() + datetime.timedelta(minutes=2)

  ext_client = k8s_client.ExtensionsV1beta1Api(api_client)

  while datetime.datetime.now() < end_time:
    deploy = ext_client.read_namespaced_deployment("tiller-deploy", "kube-system")
    if deploy.status.ready_replicas >= 1:
      logging.info("tiller is ready")
      return
    logging.info("Waiting for tiller")
    time.sleep(10)

  raise ValueError("Timeout waiting for tiller")

def install_gpu_drivers(api_client):
  """Install GPU drivers on the cluster.

  Note: GPU support in K8s is very much Alpha and this code will
  likely change quite frequently.

  Return:
     ds: Daemonset for the GPU installer
  """
  logging.info("Install GPU Drivers.")
  # Fetch the daemonset to install the drivers.
  link = "https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/k8s-1.8/device-plugin-daemonset.yaml"  # pylint: disable=line-too-long
  f = urllib.urlopen(link)
  daemonset_spec = yaml.load(f)
  ext_client = k8s_client.ExtensionsV1beta1Api(api_client)
  try:
    namespace = daemonset_spec["metadata"]["namespace"]
    ext_client.create_namespaced_daemon_set(namespace, daemonset_spec)
  except rest.ApiException as e:
    # Status appears to be a string.
    if e.status == 409:
      logging.info("GPU driver daemon set has already been installed")
    else:
      raise

def wait_for_gpu_driver_install(api_client,
                                timeout=datetime.timedelta(minutes=10)):
  """Wait until some nodes are available with GPUs."""

  end_time = datetime.datetime.now() + timeout
  api = k8s_client.CoreV1Api(api_client)
  while datetime.datetime.now() <= end_time:
    nodes = api.list_node()
    for n in nodes.items:
      if n.status.capacity.get("nvidia.com/gpu", 0) > 0:
        logging.info("GPUs are available.")
        return
    logging.info("Waiting for GPUs to be ready.")
    time.sleep(15)
  logging.error("Timeout waiting for GPU nodes to be ready.")
  raise TimeoutError("Timeout waiting for GPU nodes to be ready.")

def cluster_has_gpu_nodes(api_client):
  """Return true if the cluster has nodes with GPUs."""
  api = k8s_client.CoreV1Api(api_client)
  nodes = api.list_node()

  for n in nodes.items:
    if "cloud.google.com/gke-accelerator" in n.metadata.labels:
      return True
  return False

def create_tiller_service_accounts(api_client):
  logging.info("Creating service account for tiller.")
  api = k8s_client.CoreV1Api(api_client)
  body = yaml.load("""apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system""")
  try:
    api.create_namespaced_service_account("kube-system", body)
  except rest.ApiException as e:
    if e.status == 409:
      logging.info("Service account tiller already exists.")
    else:
      raise
  body = yaml.load("""apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
""")
  rbac_api = k8s_client.RbacAuthorizationV1beta1Api(api_client)
  try:
    rbac_api.create_cluster_role_binding(body)
  except rest.ApiException as e:
    if e.status == 409:
      logging.info("Role binding for service account tiller already exists.")
    else:
      raise

def setup_cluster(api_client):
  """Setup a cluster.

  This function assumes kubectl has already been configured to talk to your
  cluster.

  Args:
    use_gpus
  """
  create_tiller_service_accounts(api_client)
  run(["helm", "init", "--service-account=tiller"])
  use_gpus = cluster_has_gpu_nodes(api_client)
  if use_gpus:
    logging.info("GPUs detected in cluster.")
  else:
    logging.info("No GPUs detected in cluster.")

  if use_gpus:
    install_gpu_drivers(api_client)
  wait_for_tiller_to_be_ready(api_client)
  if use_gpus:
    wait_for_gpu_driver_install(api_client)

class TimeoutError(Exception):
  """An error indicating an operation timed out."""

GCS_REGEX = re.compile("gs://([^/]*)/(.*)")

def split_gcs_uri(gcs_uri):
  """Split a GCS URI into bucket and path."""
  m = GCS_REGEX.match(gcs_uri)
  bucket = m.group(1)
  path = m.group(2)
  return bucket, path

def _refresh_credentials():
  # I tried userinfo.email scope that was insufficient; got unauthorized errors.
  credentials, _ = google.auth.default(
    scopes=["https://www.googleapis.com/auth/cloud-platform"])
  request = google.auth.transport.requests.Request()
  credentials.refresh(request)
  return credentials

# TODO(jlewi): This is a work around for
# https://github.com/kubernetes-incubator/client-python/issues/339.
# Consider getting rid of this and adopting the solution to that issue.
def load_kube_config(config_file=None, context=None,
                     client_configuration=configuration,
                     persist_config=True,
                     get_google_credentials=_refresh_credentials,
                     **kwargs):
  """Loads authentication and cluster information from kube-config file
  and stores them in kubernetes.client.configuration.

  :param config_file: Name of the kube-config file.
  :param context: set the active context. If is set to None, current_context
      from config file will be used.
  :param client_configuration: The kubernetes.client.ConfigurationObject to
      set configs to.
  :param persist_config: If True, config file will be updated when changed
      (e.g GCP token refresh).
  """

  if config_file is None:
    config_file = os.path.expanduser(kube_config.KUBE_CONFIG_DEFAULT_LOCATION)

  config_persister = None
  if persist_config:
    def _save_kube_config(config_map):
      with open(config_file, 'w') as f:
        yaml.safe_dump(config_map, f, default_flow_style=False)
    config_persister = _save_kube_config

  kube_config._get_kube_config_loader_for_yaml_file(  # pylint: disable=protected-access
    config_file, active_context=context,
      client_configuration=client_configuration,
        config_persister=config_persister,
        get_google_credentials=get_google_credentials,
        **kwargs).load_and_set()
