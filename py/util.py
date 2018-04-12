"""Utilities used by our python scripts for building and releasing."""
from __future__ import print_function

# TODO(jlewi): We should remove this file and use util in kubeflow/testing repository.
import datetime
import logging
import os
import re
import subprocess
import tempfile
import time
import urllib
import yaml

import google.auth
import google.auth.transport
import google.auth.transport.requests

from googleapiclient import errors
from kubernetes import client as k8s_client
from kubernetes.config import kube_config
from kubernetes.client import configuration as kubernetes_configuration
from kubernetes.client import rest

# Default name for the repo organization and name.
# This should match the values used in Go imports.
# We default to environment variables so that it can be set correctly when
# running under prow.
MASTER_REPO_OWNER = os.getenv("REPO_OWNER", "kubeflow")
MASTER_REPO_NAME = os.getenv("REPO_NAME", "tf-operator")


# TODO(jlewi): Should we stream the output by polling the subprocess?
# look at run_and_stream in build_and_push.
#
# TODO(jlewi): I think we can delete use_print after updating callers. that
# was a hack to make output show up in Airflow; I think writing subprocess
# to a file and then logging it works better.
def run(command, cwd=None, env=None, dryrun=False):
  """Run a subprocess.

  Any subprocess output is emitted through the logging modules.
  """
  logging.info("Running: %s \ncwd=%s", " ".join(command), cwd)

  # In case the release is done from a non-linux machine
  # we enforce correct GOOS and GOARCH
  extra_envs = {
    "GOOS": "linux",
    "GOARCH": "amd64"
    }

  if not env:
    env = os.environ.copy()
    env.update(extra_envs)
  else:
    env.update(extra_envs)
    keys = sorted(env.keys())

    lines = []
    for k in keys:
      lines.append("{0}={1}".format(k, env[k]))
    logging.info("Running: Environment:\n%s", "\n".join(lines))

  log_file = None
  try:
    if dryrun:
      command_str = ("Dryrun: Command:\n{0}\nCWD:\n{1}\n"
                     "Environment:\n{2}").format(" ".join(command), cwd, env)
      logging.info(command_str)

    # We write stderr/stdout to a file and then read it and process it.
    # We do this because if just inherit the handles from the parent the
    # subprocess output doesn't show up in Airflow. This might be because
    # we had multiple levels of processes invoking python processes.
    with tempfile.NamedTemporaryFile(
        prefix="tmpRunLogs", delete=False, mode="w") as hf:
      log_file = hf.name
      subprocess.check_call(command, cwd=cwd, env=env, stdout=hf, stderr=hf)
  finally:
    with open(log_file, "r") as hf:
      output = hf.read()
    logging.info("Subprocess output:\n%s", output)


def run_and_output(command, cwd=None, env=None):
  logging.info("Running: %s \ncwd=%s", " ".join(command), cwd)

  if not env:
    env = os.environ
  # The output won't be available until the command completes.
  # So prefer using run if we don't need to return the output.
  try:
    output = subprocess.check_output(
      command, cwd=cwd, env=env, stderr=subprocess.STDOUT).decode("utf-8")
    logging.info("Subprocess output:\n%s", output)
  except subprocess.CalledProcessError as e:
    logging.info("Subprocess output:\n%s", e.output)
    raise
  return output


def clone_repo(dest,
               repo_owner=MASTER_REPO_OWNER,
               repo_name=MASTER_REPO_NAME,
               sha=None,
               branches=None):
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
      run(
        [
          "git",
          "fetch",
          "origin",
          b,
        ], cwd=dest)

    if not sha:
      b = branches[-1].split(":", 1)[-1]
      run(
        [
          "git",
          "checkout",
          b,
        ], cwd=dest)

  if sha:
    run(["git", "checkout", sha], cwd=dest)

  # Get the actual git hash.
  # This ensures even for periodic jobs which don't set the sha we know
  # the version of the code tested.
  sha = run_and_output(["git", "rev-parse", "HEAD"], cwd=dest)

  return dest, sha


def install_go_deps(src_dir):
  """Run glide to install dependencies."""
  # Install dependencies
  run(["glide", "install", "--strip-vendor"], cwd=src_dir)


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
  request = gke.projects().zones().clusters().create(
    body=cluster_request, projectId=project, zone=zone)

  try:
    logging.info("Creating cluster; project=%s, zone=%s, name=%s", project,
                 zone, cluster_request["cluster"]["name"])
    response = request.execute()
    logging.info("Response %s", response)
    create_op = wait_for_operation(gke, project, zone, response["name"])
    logging.info("Cluster creation done.\n %s", create_op)

  except errors.HttpError as e:
    logging.error("Exception occured creating cluster: %s, status: %s", e,
                  e.resp["status"])
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

  request = gke.projects().zones().clusters().delete(
    clusterId=name, projectId=project, zone=zone)

  try:
    response = request.execute()
    logging.info("Response %s", response)
    delete_op = wait_for_operation(gke, project, zone, response["name"])
    logging.info("Cluster deletion done.\n %s", delete_op)

  except errors.HttpError as e:
    logging.error("Exception occured deleting cluster: %s, status: %s", e,
                  e.resp["status"])


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
        projectId=project, zone=zone, operationId=op_id).execute()
    else:
      op = client.globalOperations().get(
        project=project, operation=op_id).execute()

    status = op.get("status", "")
    # Need to handle other status's
    if status == "DONE":
      return op
    if datetime.datetime.now() > endtime:
      raise TimeoutError(
        "Timed out waiting for op: {0} to complete.".format(op_id))
    time.sleep(polling_interval.total_seconds())

  # Linter complains if we don't have a return here even though its unreachable.
  return None


def configure_kubectl(project, zone, cluster_name):
  logging.info("Configuring kubectl")
  run([
    "gcloud", "--project=" + project, "container", "clusters", "--zone=" + zone,
    "get-credentials", cluster_name
  ])


def wait_for_deployment(api_client, namespace, name):
  """Wait for deployment to be ready.

  Args:
    api_client: K8s api client to use.
    namespace: The name space for the deployment.
    name: The name of the deployment.

  Returns:
    deploy: The deploy object describing the deployment.

  Raises:
    TimeoutError: If timeout waiting for deployment to be ready.
  """
  # Wait for deployment to be ready
  end_time = datetime.datetime.now() + datetime.timedelta(minutes=2)

  ext_client = k8s_client.ExtensionsV1beta1Api(api_client)

  while datetime.datetime.now() < end_time:
    deploy = ext_client.read_namespaced_deployment(name, namespace)
    if deploy.status.ready_replicas >= 1:
      logging.info("Deployment %s in namespace %s is ready", name, namespace)
      return deploy
    logging.info("Waiting for deployment %s in namespace %s", name, namespace)
    time.sleep(10)

  logging.error("Timeout waiting for deployment %s in namespace %s to be "
                "ready", name, namespace)
  raise TimeoutError(
    "Timeout waiting for deployment {0} in namespace {1}".format(
      name, namespace))


def wait_for_statefulset(api_client, namespace, name):
  """Wait for deployment to be ready.

  Args:
    api_client: K8s api client to use.
    namespace: The name space for the deployment.
    name: The name of the stateful set.

  Returns:
    deploy: The deploy object describing the deployment.

  Raises:
    TimeoutError: If timeout waiting for deployment to be ready.
  """
  # Wait for tiller to be ready
  end_time = datetime.datetime.now() + datetime.timedelta(minutes=2)

  apps_client = k8s_client.AppsV1beta1Api(api_client)

  while datetime.datetime.now() < end_time:
    stateful = apps_client.read_namespaced_stateful_set(name, namespace)
    if stateful.status.ready_replicas >= 1:
      logging.info("Statefulset %s in namespace %s is ready", name, namespace)
      return stateful
    logging.info("Waiting for Statefulset %s in namespace %s", name, namespace)
    time.sleep(10)

  logging.error("Timeout waiting for statefulset %s in namespace %s to be "
                "ready", name, namespace)
  raise TimeoutError(
    "Timeout waiting for statefulset {0} in namespace {1}".format(
      name, namespace))


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


def setup_cluster(api_client):
  """Setup a cluster.

  This function assumes kubectl has already been configured to talk to your
  cluster.

  Args:
    use_gpus
  """
  use_gpus = cluster_has_gpu_nodes(api_client)
  if use_gpus:
    logging.info("GPUs detected in cluster.")
  else:
    logging.info("No GPUs detected in cluster.")

  if use_gpus:
    install_gpu_drivers(api_client)
  if use_gpus:
    wait_for_gpu_driver_install(api_client)


# TODO(jlewi): TimeoutError is a built in exception in python3 so we can
# delete this when we go to Python3.
class TimeoutError(Exception):  # pylint: disable=redefined-builtin
  """An error indicating an operation timed out."""


GCS_REGEX = re.compile("gs://([^/]*)(/.*)?")


def split_gcs_uri(gcs_uri):
  """Split a GCS URI into bucket and path."""
  m = GCS_REGEX.match(gcs_uri)
  bucket = m.group(1)
  path = ""
  if m.group(2):
    path = m.group(2).lstrip("/")
  return bucket, path


def _refresh_credentials():
  # userinfo.email scope was insufficient for authorizing requests to K8s.
  credentials, _ = google.auth.default(
    scopes=["https://www.googleapis.com/auth/cloud-platform"])
  request = google.auth.transport.requests.Request()
  credentials.refresh(request)
  return credentials


# TODO(jlewi): This is a work around for
# https://github.com/kubernetes-incubator/client-python/issues/339.
# Consider getting rid of this and adopting the solution to that issue.
#
# This function is based on
# https://github.com/kubernetes-client/python-base/blob/master/config/kube_config.py#L331
# we modify it though so that we can pass through the function to get credentials.
def load_kube_config(config_file=None,
                     context=None,
                     client_configuration=None,
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

  loader = kube_config._get_kube_config_loader_for_yaml_file(  # pylint: disable=protected-access
    config_file,
    active_context=context,
    config_persister=config_persister,
    get_google_credentials=get_google_credentials,
    **kwargs)

  if client_configuration is None:
    config = type.__call__(kubernetes_configuration.Configuration)
    loader.load_and_set(config)
    kubernetes_configuration.Configuration.set_default(config)
  else:
    loader.load_and_set(client_configuration)


def maybe_activate_service_account():
  if os.getenv("GOOGLE_APPLICATION_CREDENTIALS"):
    logging.info("GOOGLE_APPLICATION_CREDENTIALS is set; configuring gcloud "
                 "to use service account.")
    run([
      "gcloud", "auth", "activate-service-account",
      "--key-file=" + os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
    ])
