"""Utilities used by our python scripts for building and releasing."""
from __future__ import print_function

import logging
import os
import shutil
import subprocess

# Default name for the repo organization and name.
# This should match the values used in Go imports.
MASTER_REPO_OWNER = "jlewi"
MASTER_REPO_NAME = "mlkube.io"


def run(command, cwd=None):
  """Run a subprocess.

  Any subprocess output is emitted through the logging modules.
  """
  logging.info("Running: %s \ncwd=%s", " ".join(command), cwd)

  try:
    output = subprocess.check_output(command, cwd=cwd)
    logging.info("Subprocess output:\n%s", output)
  except subprocess.CalledProcessError as e:
    logging.info("Subprocess output:\n%s", e.output)
    raise

def run_and_output(command, cwd=None):
  logging.info("Running: %s \ncwd=%s", " ".join(command), cwd)
  # The output won't be available until the command completes.
  # So prefer using run if we don't need to return the output.
  try:
    output = subprocess.check_output(command, cwd=cwd).decode("utf-8")
    logging.info("Subprocess output:\n%s", output)
  except subprocess.CalledProcessError as e:
    logging.info("Subprocess output:\n%s", e.output)
    raise
  return output


def clone_repo(dest, repo_owner=MASTER_REPO_OWNER, repo_name=MASTER_REPO_NAME,
               sha=None):
  """Clone the repo,

  Returns:
    dest: This is the root path for the training code.
    repo_owner: The owner for github organization.
    repo_name: The repo name.
    sha: The sha number of the repo.
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
