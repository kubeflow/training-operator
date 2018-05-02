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
import argparse
import json
import logging
import os
import re
import time

from google.cloud import storage  # pylint: disable=no-name-in-module
from py import test_util
from py import util  # pylint: disable=no-name-in-module

# Default repository organization and name.
# This should match the values used in Go imports.
GO_REPO_OWNER = "kubeflow"
GO_REPO_NAME = "tf-operator"

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
                owner=GO_REPO_OWNER,
                repo=GO_REPO_NAME,
                pull_number=pull_number,
                job=job_name,
                build=os.getenv("BUILD_NUMBER"))
    return output
  elif os.getenv("REPO_OWNER"):
    # It is a postsubmit job
    output = ("gs://kubernetes-jenkins/logs/{owner}_{repo}/"
              "{job}/{build}").format(
                owner=GO_REPO_OWNER,
                repo=GO_REPO_NAME,
                job=job_name,
                build=os.getenv("BUILD_NUMBER"))
    return output

  # Its a periodic job
  output = ("gs://kubernetes-jenkins/logs/{job}/{build}").format(
    job=job_name, build=os.getenv("BUILD_NUMBER"))
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
              job=job_name, build=build_number)
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
      GO_REPO_OWNER + "/" + GO_REPO_NAME:
      sha,
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
    symlink: GCS path of the object to server as the link
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
  bucket_name = "kubeflow-ci-results"
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


def _get_actual_junit_files(bucket, prefix):
  actual_junit = set()
  for b in bucket.list_blobs(prefix=os.path.join(prefix, "junit")):
    actual_junit.add(os.path.basename(b.name))
  return actual_junit


def check_no_errors(gcs_client, artifacts_dir, junit_files):
  """Check that all the XML files exist and there were no errors.

  Args:
    gcs_client: The GCS client.
    artifacts_dir: The directory where artifacts should be stored.
    junit_files: List of the names of the junit files.

  Returns:
    True if there were no errors and false otherwise.
  """
  bucket_name, prefix = util.split_gcs_uri(artifacts_dir)
  bucket = gcs_client.get_bucket(bucket_name)
  no_errors = True

  # Get a list of actual junit files.
  actual_junit = _get_actual_junit_files(bucket, prefix)

  for f in junit_files:
    full_path = os.path.join(artifacts_dir, f)
    logging.info("Checking %s", full_path)
    b = bucket.blob(os.path.join(prefix, f))
    if not b.exists():
      logging.error("Missing %s", full_path)
      no_errors = False
      continue

    xml_contents = b.download_as_string()

    if test_util.get_num_failures(xml_contents) > 0:
      logging.info("Test failures in %s", full_path)
      no_errors = False

  # Check if there were any extra tests that ran and treat
  # that as a failure.
  extra = set(actual_junit) - set(junit_files)
  if extra:
    logging.error("Extra junit files found: %s", ",".join(extra))
    no_errors = False
  return no_errors


def finalize_prow_job(args):
  """Finalize a prow job.

  Finalizing a PROW job consists of determining the status of the
  prow job by looking at the junit files and then creating finished.json.
  """
  junit_files = args.junit_files.split(",")
  gcs_client = storage.Client()

  output_dir = get_gcs_output()
  artifacts_dir = os.path.join(output_dir, "artifacts")
  no_errors = check_no_errors(gcs_client, artifacts_dir, junit_files)

  create_finished(gcs_client, output_dir, no_errors)


def main():  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals
  logging.basicConfig(
    level=logging.INFO,
    format=('%(levelname)s|%(asctime)s'
            '|%(pathname)s|%(lineno)d| %(message)s'),
    datefmt='%Y-%m-%dT%H:%M:%S',
  )

  # create the top-level parser
  parser = argparse.ArgumentParser(description="Steps related to prow.")
  subparsers = parser.add_subparsers()

  #############################################################################
  # Finalize prow job.
  parser_finished = subparsers.add_parser(
    "finalize_job", help="Finalize the prow job.")

  parser_finished.add_argument(
    "--junit_files",
    default="",
    type=str,
    help=("A comma separated list of the names of "
          "the expected junit files."))
  parser_finished.set_defaults(func=finalize_prow_job)

  # parse the args and call whatever function was selected
  args = parser.parse_args()

  args.func(args)


if __name__ == "__main__":
  main()
