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
"""
import logging
import subprocess
import os
import shutil

# Default name for the repo and name.
# This should match the values used in Go imports.
GO_REPO_OWNER = "jlewi"
GO_REPO_NAME = "mlkube.io"

def run(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  subprocess.check_call(command, cwd=cwd)

def clone_repo():
  go_path = os.getenv("GOPATH")
  # REPO_OWNER and REPO_NAME are the environment variables set by Prow.
  repo_owner = os.getenv("REPO_OWNER")
  repo_name = os.getenv("REPO_NAME")

  if bool(repo_owner) != bool(repo_name):
    raise ValueError("Either set both environment variables "
                     "REPO_OWNER and REPO_NAME or set neither.")

  src_dir = os.path.join(go_path, "src/github.com", GO_REPO_OWNER)
  dest = os.path.join(src_dir, GO_REPO_NAME)

  if not repo_owner and not_repo_name:
    logging.info("Environment variables REPO_OWNER and REPO_NAME not set; "
                 "not checking out code.")
    if not os.path.exists(dest):
      raise ValueError("No code found at %s", dest)
    return

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

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)

  parser = argparse.ArgumentParser(
    description="Run E2E tests for the TfJob CRD.")

  parser.add_argument(
    "--project",
      default="mlkube-testing",
      type=str,
      help="Google project to use for GCR and GKE.")

  args = parser.parse_args()

  # Activate the service account for gcloud
  # If you don't activate it then you should already be logged in.
  if os.getenv("GOOGLE_APPLICATION_CREDENTIALS"):
    logging.info("GOOGLE_APPLICATION_CREDENTIALS=%s",
                 os.getenv("GOOGLE_APPLICATION_CREDENTIALS"))
    run(["gcloud", "auth", "activate-service-account",
         "--key-file={0}".format(os.getenv("GOOGLE_APPLICATION_CREDENTIALS"))])

  clone_repo()

  # Build and push the image
  # We use Google Container Builder because Prow currently doesn't allow using
  # docker build.
  # TODO(jlewi): Add an option to not build with GCB. This will be convenient
  # for running the test on local changes.
  registry = "gcr.io/" + args.project
  run(["./images/tf_operator/build_and_push.py", "--gcb",
       "--project=" + args.project,
       "--registry=gcr.io/mlkube-testing"], cwd=dest)

  # Create a GKE cluster.