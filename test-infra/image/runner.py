#!/usr/bin/python
# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Run an E2E test.

An E2E test consists of the following steps

1. Clone code from github
  * This step is skipped if environment variables specifiying the repo aren't
    set.
  * In this case the code should already be mounted inside the container.
  * Skipping cloning the repo is useful if you want to run an E2E test
    using your local changes.

2. Build and push a Docker image for the CRD.

TODO(jlewi): Will we be able to eventually replace this with the bootstrap
program in https://github.com/kubernetes/test-infra/tree/master/bootstrap
"""
import argparse
import datetime
import json
import logging
import subprocess
import os
import re
import shutil
import tempfile
import time
import uuid
import yaml

from googleapiclient import discovery
from googleapiclient import errors
from oauth2client.client import GoogleCredentials
from google.cloud import storage

# Default name for the repo and name.
# This should match the values used in Go imports.
GO_REPO_OWNER = "jlewi"
GO_REPO_NAME = "mlkube.io"

GCS_REGEX = re.compile("gs://([^/]*)/(.*)")

def run(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  subprocess.check_call(command, cwd=cwd)

def clone_repo():
  """Clone the repo.

  Returns:
    src_path: This is the root path for the training code.
    sha: The sha number of the repo.
  """
  go_path = os.getenv("GOPATH")
  # REPO_OWNER and REPO_NAME are the environment variables set by Prow.
  repo_owner = os.getenv("REPO_OWNER")
  repo_name = os.getenv("REPO_NAME")

  if bool(repo_owner) != bool(repo_name):
    raise ValueError("Either set both environment variables "
                     "REPO_OWNER and REPO_NAME or set neither.")

  src_dir = os.path.join(go_path, "src/github.com", GO_REPO_OWNER)
  dest = os.path.join(src_dir, GO_REPO_NAME)

  if not repo_owner and not repo_name:
    logging.info("Environment variables REPO_OWNER and REPO_NAME not set; "
                 "not checking out code.")
    if not os.path.exists(dest):
      raise ValueError("No code found at %s", dest)
    # TODO(jlewi): We should get the sha number and add "-dirty" if needed
    return dest, ""

  # Clone mlkube
  repo = "https://github.com/{0}/{1}.git".format(repo_owner, repo_name)
  logging.info("repo %s", repo)

  os.makedirs(src_dir)

  # TODO(jlewi): How can we figure out what branch
  run(["git", "clone",  repo, dest])

  # If this is a presubmit PULL_PULL_SHA will be set
  # see:
  # https://github.com/kubernetes/test-infra/tree/master/prow#job-evironment-variables
  sha = os.getenv('PULL_PULL_SHA')

  if not sha:
    # For postsubmits PULL_BASE_SHA will be set.
    sha = os.getenv('PULL_BASE_SHA')

  if sha:
    run(["git", "checkout", sha], cwd=dest)

  # Install dependencies
  run(["glide", "install"], cwd=dest)

  # We need to remove the vendored dependencies of apiextensions otherwise we
  # get type conflicts.
  shutil.rmtree(os.path.join(dest,
                             "vendor/k8s.io/apiextensions-apiserver/vendor"))

  return dest, sha

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

def create_cluster(gke, name, project, zone):
  """Create the cluster.

  Args:
    gke: Client for GKE.

  """
  cluster_request = {
    "cluster": {
       "name": args.cluster,
          "description": "A GKE cluster for testing GPUs with Cloud ML",
          "initialNodeCount": 1,
          "nodeConfig": {
            "machineType": "n1-standard-8",
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
      # TODO(jlewi): What should we do if the cluster already exits?
      pass
    else:
      raise

  logging.info("Configuring kubectl")
  run(["gcloud", "--project=" + project, "container",
       "clusters", "--zone=" + zone, "get-credentials", name])

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

def build_container(use_gcb, src_dir, test_dir):
  """Build the CRD container.

  Args:
    use_gcb: Boolean indicating whether to build the image with GCB or Docker.
    src_dir: The directory containing the source.
    test_dir: Scratch directory for runner.py.

  Returns:
    image: The URI of the newly built image.
  """
  # Build and push the image
  # We use Google Container Builder because Prow currently doesn't allow using
  # docker build.
  # TODO(jlewi): Add an option to not build with GCB. This will be convenient
  # for running the test on local changes.
  registry = "gcr.io/" + args.project
  if use_gcb:
    gcb_arg = "--gcb"
  else:
    gcb_arg = "--no-gcb"

  build_info_file = os.path.join(test_dir, "build_info.yaml")
  run(["./images/tf_operator/build_and_push.py", gcb_arg,
       "--project=" + args.project,
       "--registry=gcr.io/mlkube-testing",
       "--output=" + build_info_file], cwd=src_dir)

  with open(build_info_file) as hf:
    build_info = yaml.load(hf)

  return build_info["image"]

def deploy_and_test(image, test_dir):
  """Deploy and test the CRD.

  Args:
    image: The Docker image for the CRD to use.
    test_dir: The directory where test outputs should be written.

  Returns:
    success: Boolean indicating success or failure
  """

  target = os.path.join("github.com", GO_REPO_OWNER, GO_REPO_NAME,
                        "test-infra", "helm-test")
  run(["go", "install", target])

  binary = os.path.join(os.getenv("GOPATH"), "bin", "helm-test")
  try:
    run([binary, "--image=" + image, "--output_dir=" + test_dir])
  except subprocess.CalledProcessError as e:
    logging.error("helm-test failed; %s", e)
    return False
  return True

def get_gcs_output():
  """Return the GCS directory where test outputs should be."""
  job_name = os.getenv("JOB_NAME")

  # GCS layout is defined here:
  # https://github.com/kubernetes/test-infra/tree/master/gubernator#job-artifact-gcs-layout
  pull_number = os.getenv("PULL_NUMBER")
  if pull_number:
    output = ("gs://kubernetes-jenkins/pr-logs/pull/{owner}_{repo}/"
              "{pull_number}/{job}/{build}").format(
                owner=GO_REPO_OWNER, repo=GO_REPO_NAME,
                pull_number=pull_number,
                job=job_name,
                build=os.getenv("BUILD_NUMBER"))
    return output
  else:
    # Its a periodic or postsubmit job
    output = ("gs://kubernetes-jenkins/logs/"
              "{job}/{build}").format(
              owner=GO_REPO_OWNER, repo=GO_REPO_NAME,
              pull_number=pull_number,
              job=job_name,
              build=os.getenv("BUILD_NUMBER"))
    return output

def create_started(gcs_client, output_dir, sha):
  """Create the started output in GCS.

  Args:
    gcs_client: GCS client
    output_dir: The GCS directory where the output should be written.
    sha: Sha for the mlkube.io repo
  """
  # See https://github.com/kubernetes/test-infra/tree/master/gubernator#job-artifact-gcs-layout
  # For a list of fields expected by gubernator
  started = {
    "timestamp": int(time.time()),
    "pull": os.getenv("PULL_REFS", ""),
    "repos": {
        # List all repos used and their versions.
        GO_REPO_OWNER + "/" + GO_REPO_NAME: sha,
     },
  }

  m = GCS_REGEX.match(output_dir)
  bucket = m.group(1)
  path = m.group(2)

  bucket = gcs_client.get_bucket(bucket)
  blob = bucket.blob(os.path.join(path, "started.json"))
  blob.upload_from_string(json.dumps(started))

def create_finished(gcs_client, output_dir, success):
  """Create the finished output in GCS.

  Args:
    gcs_client: GCS client
    output_dir: The GCS directory where the output should be written.
    success: Boolean indicating whether the test was successful.
  """
  result = "FAILURE"
  if success:
    result = "SUCCESS"
  finished = {
    "timestamp": int(time.time()),
    "result": result,
    # Dictionary of extra key value pairs to display to the user.
    "metadata": {},
  }

  m = GCS_REGEX.match(output_dir)
  bucket = m.group(1)
  path = m.group(2)

  bucket = gcs_client.get_bucket(bucket)
  blob = bucket.blob(os.path.join(path, "finished.json"))
  blob.upload_from_string(json.dumps(finished))

def upload_outputs(gcs_client, output_dir, test_dir):
  # TODO(jlewi): We need to copy the _artifacts dir to GCS
  # See: https://github.com/kubernetes/test-infra/tree/master/gubernator#job-artifact-gcs-layout
  # test_dir mentioned above ends up looking like
  #ls -la /tmp/tmpTfCrdTestyELzn1/
  #total 16
  #drwx------  2 root root 4096 Oct 16 21:32 .
  #drwxrwxrwt 17 root root 4096 Oct 16 21:28 ..
  #-rw-r--r--  1 root root   77 Oct 16 21:28 build_info.yaml
  #-rw-r--r--  1 root root  722 Oct 16 21:32 junit_01.xml
  # Example artifacts directory for a presubmit job
  # gs://kubernetes-jenkins/pr-logs/pull/jlewi_mlkube.io/49/mlkube-build-presubmit/12

  # TODO(jlewi): DO NOT SUBMIT. We only want to leave cluster up to
  # facilitate debugging the test.
  m = GCS_REGEX.match(output_dir)
  bucket = m.group(1)
  path = m.group(2)

  bucket = gcs_client.get_bucket(bucket)

  build_file = os.path.join(test_dir, "build_info.yaml")
  if not os.patth.exists(build_file):
    logging.error("File %s doesn't exist.", build_file)
  else:
    logging.info("Uploading file %s.", build_file)
    blob = bucket.blob(os.path.join(path, "build_info.yaml"))
    blob.upload_from_filename(build_file)

  junit_file = os.path.join(test_dir, "junit_01.xml")
  if not os.patth.exists(junit_file):
    logging.error("File %s doesn't exist.", junit_file)
  else:
    logging.info("Uploading file %s.", junit_file)
    blob = bucket.blob(os.path.join(path, "artifacts", "junit_01.xml"))
    blob.upload_from_filename(junit_file)


if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)

  parser = argparse.ArgumentParser(
    description="Run E2E tests for the TfJob CRD.")

  parser.add_argument(
    "--project",
    default="mlkube-testing",
      type=str,
      help="Google project to use for GCR and GKE.")

  n = datetime.datetime.now()

  parser.add_argument(
    "--cluster",
    default=n.strftime("v%Y%m%d") + "-" + uuid.uuid4().hex[0:4],
      type=str,
      help="Name for the cluster")

  parser.add_argument(
    "--zone",
    default="us-central1-f",
      type=str,
      help="Zone to use for spinning up the GKE cluster.")


  parser.add_argument("--gcb", dest="use_gcb", action="store_true",
                      help="Use Google Container Builder to build the image.")
  parser.add_argument("--no-gcb", dest="use_gcb", action="store_false",
                      help="Use Docker to build the image.")
  parser.set_defaults(use_gcb=True)

  parser.add_argument("--gke", dest="use_gke", action="store_true",
                      help="Use Google Container Engine to create acluster.")
  parser.add_argument("--no-gke", dest="use_gke", action="store_false",
                      help=("Do not use GKE to create a cluster. "
                            "A cluster must already exist and be accessible "
                            "with kubectl."))
  parser.set_defaults(use_gke=True)

  args = parser.parse_args()

  # Print environment variables.
  # This is useful for debugging and understanding the information set by prow.
  names = os.environ.keys()
  names.sort()
  logging.info("Environment Variables")
  for n in names:
    logging.info("%s=%s", n, os.environ[n])
  logging.info("End Environment Variables")

  test_dir = tempfile.mkdtemp(prefix="tmpTfCrdTest")
  logging.info("test_dir: %s", test_dir)

  # Activate the service account for gcloud
  # If you don't activate it then you should already be logged in.
  if os.getenv("GOOGLE_APPLICATION_CREDENTIALS"):
    logging.info("GOOGLE_APPLICATION_CREDENTIALS=%s",
                 os.getenv("GOOGLE_APPLICATION_CREDENTIALS"))
    run(["gcloud", "auth", "activate-service-account",
         "--key-file={0}".format(os.getenv("GOOGLE_APPLICATION_CREDENTIALS"))])

  src_dir, sha = clone_repo()

  output_dir = get_gcs_output()
  gcs_client = storage.Client()

  create_started(gcs_client, output_dir, sha)

  logging.info("Artifacts will be saved to: %s", output_dir)

  # TODO(jlewi): We should consider moving all the code below into a script
  # that is invoked in the src_dir. Currently runner.py is baked into
  # the Docker image invoked by the prow job. So the version of runner.py
  # doesn't match the version of the repo cloned above. If we instead move
  # it into a script inside the repo and invoke the repo then we will be
  # invoking the script at the CL we have checked out. I think that's better
  # because then presubmits automatically test any changes to the test script.
  image = build_container(args.use_gcb, src_dir, test_dir)
  logging.info("Created image: %s", image)

  credentials = GoogleCredentials.get_application_default()
  gke = discovery.build("container", "v1", credentials=credentials)

  success = False
  try:
    # Create a GKE cluster.
    create_cluster(gke, args.cluster, args.project, args.zone)

    success = deploy_and_test(image, test_dir)

  finally:
    create_finished(gcs_client, output_dir, success)
    upload_outputs(gcs_client, output_dir, test_dir)
    # delete_cluster(gke, args.cluster, args.project, args.zone)