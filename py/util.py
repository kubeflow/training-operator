"""Utilities used by our python scripts for building and releasing."""
from __future__ import print_function

import datetime
import logging
import os
import shutil
import subprocess

# Default name for the repo organization and name.
# This should match the values used in Go imports.
MASTER_REPO_OWNER = "tensorflow"
MASTER_REPO_NAME = "k8s"


def run(command, cwd=None, env=None):
  """Run a subprocess.

  Any subprocess output is emitted through the logging modules.
  """
  logging.info("Running: %s \ncwd=%s", " ".join(command), cwd)

  if not env:
    env = os.environ

  try:
    output = subprocess.check_output(command, cwd=cwd, env=env,
                                     stderr=subprocess.STDOUT)
    logging.info("Subprocess output:\n%s", output)
  except subprocess.CalledProcessError as e:
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
               sha=None):
  """Clone the repo,

  Args:
    dest: This is the root path for the training code.
    repo_owner: The owner for github organization.
    repo_name: The repo name.
    sha: The sha number of the repo.

  Returns:
    dest: Directory where it was checked out
    sha: The sha of the code.
  """
  # Clone mlkube
  repo = "https://github.com/{0}/{1}.git".format(repo_owner, repo_name)
  logging.info("repo %s", repo)

  # TODO(jlewi): How can we figure out what branch
  run(["git", "clone", repo, dest])

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