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
"""Helper functions for working with prow.
"""
import json
import logging
import os
import re
import subprocess
import time
from py import util  # pylint: disable=no-name-in-module

# Default repository organization and name.
# This should match the values used in Go imports.
GO_REPO_OWNER = "tensorflow"
GO_REPO_NAME = "k8s"

GCS_REGEX = re.compile("gs://([^/]*)/(.*)")

def get_gcs_output():
  """Return the GCS directory where test outputs should be written to."""
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
  elif os.getenv("REPO_OWNER"):
    # It is a postsubmit job
    output = ("gs://kubernetes-jenkins/logs/{owner}_{repo}/"
              "{job}/{build}").format(
                  owner=GO_REPO_OWNER, repo=GO_REPO_NAME,
                  job=job_name,
                  build=os.getenv("BUILD_NUMBER"))
    return output

  # Its a periodic job
  output = ("gs://kubernetes-jenkins/logs/{job}/{build}").format(
      job=job_name,
      build=os.getenv("BUILD_NUMBER"))
  return output


def get_symlink_output(pull_number, job_name, build_number):
  """Return the location where the symlink should be created."""
  # GCS layout is defined here:
  # https://github.com/kubernetes/test-infra/tree/master/gubernator#job-artifact-gcs-layout
  if not pull_number:
    # Symlinks are only created for pull requests.
    return ""
  output = ("gs://kubernetes-jenkins/pr-logs/directory/"
            "{job}/{build}.txt").format(
                job=job_name,
                build=build_number)
  return output


def create_started(gcs_client, output_dir, sha):
  """Create the started output in GCS.

  Args:
    gcs_client: GCS client
    output_dir: The GCS directory where the output should be written.
    sha: Sha for the mlkube.io repo

  Returns:
    blob: The created blob.
  """
  # See:
  # https://github.com/kubernetes/test-infra/tree/master/gubernator#job-artifact-gcs-layout
  # For a list of fields expected by gubernator
  started = {
      "timestamp": int(time.time()),
      "repos": {
          # List all repos used and their versions.
          GO_REPO_OWNER + "/" + GO_REPO_NAME: sha,
      },
  }

  PULL_REFS = os.getenv("PULL_REFS", "")
  if PULL_REFS:
    started["pull"] = PULL_REFS

  m = GCS_REGEX.match(output_dir)
  bucket = m.group(1)
  path = m.group(2)

  bucket = gcs_client.get_bucket(bucket)
  blob = bucket.blob(os.path.join(path, "started.json"))
  blob.upload_from_string(json.dumps(started))

  return blob


def create_finished(gcs_client, output_dir, success):
  """Create the finished output in GCS.

  Args:
    gcs_client: GCS client
    output_dir: The GCS directory where the output should be written.
    success: Boolean indicating whether the test was successful.

  Returns:
    blob: The blob object that we created.
  """
  result = "FAILURE"
  if success:
    result = "SUCCESS"
  finished = {
      "timestamp": int(time.time()),
      "result": result,
      # Dictionary of extra key value pairs to display to the user.
      # TODO(jlewi): Perhaps we should add the GCR path of the Docker image
      # we are running in. We'd have to plumb this in from bootstrap.
      "metadata": {},
  }

  m = GCS_REGEX.match(output_dir)
  bucket = m.group(1)
  path = m.group(2)

  bucket = gcs_client.get_bucket(bucket)
  blob = bucket.blob(os.path.join(path, "finished.json"))
  blob.upload_from_string(json.dumps(finished))
  return blob


def create_symlink(gcs_client, symlink, output):
  """Create a 'symlink' to the output directory.

  Args:
    gcs_client: GCS client
    symling: GCS path of the object to server as the link
    output: The location to point to.
  """
  m = GCS_REGEX.match(symlink)
  bucket = m.group(1)
  path = m.group(2)

  bucket = gcs_client.get_bucket(bucket)
  blob = bucket.blob(path)
  blob.upload_from_string(output)
  return blob


def upload_outputs(gcs_client, output_dir, build_log):
  bucket_name, path = util.split_gcs_uri(output_dir)

  bucket = gcs_client.get_bucket(bucket_name)

  if not os.path.exists(build_log):
    logging.error("File %s doesn't exist.", build_log)
  else:
    logging.info("Uploading file %s.", build_log)
    blob = bucket.blob(os.path.join(path, "build-log.txt"))
    blob.upload_from_filename(build_log)

def get_commit_from_env():
  """Get the commit to test from prow environment variables."""
  # If this is a presubmit PULL_PULL_SHA will be set see:
  # https://github.com/kubernetes/test-infra/tree/master/prow#job-evironment-variables
  sha = ""
  pull_number = os.getenv("PULL_NUMBER", "")

  if pull_number:
    sha = os.getenv("PULL_PULL_SHA", "")
  else:
    sha = os.getenv("PULL_BASE_SHA", "")

  return sha

def create_latest(gcs_client, job_name, sha):
  """Create a file in GCS with information about the latest passing postsubmit.
  """
  bucket_name = "mlkube-testing-results"
  path = os.path.join(job_name, "latest_green.json")

  bucket = gcs_client.get_bucket(bucket_name)

  logging.info("Creating GCS output: bucket: %s, path: %s.", bucket_name, path)

  data = {
      "status": "passing",
      "job": job_name,
      "sha": sha,
  }
  blob = bucket.blob(path)
  blob.upload_from_string(json.dumps(data))


# TODO(jlewi): This function should probably be moved.
def run_lint(src_dir):
  """Run lint.

  Args:
    src_dir: the directory containing the source.

  Returns:
    success: Boolean indicating success or failure
  """
  try:
    util.run(["./lint.sh"], cwd=src_dir)
  except subprocess.CalledProcessError as e:
    logging.error("Lint checks failed; %s", e)
    return False
  return True
