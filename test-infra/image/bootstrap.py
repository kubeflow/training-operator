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
"""Bootstrap an E2E test.

We bootstrap a test by checking out the repository. Once we've checked out
the repository we can invoke the E2E test runner.

TODO(jlewi): Will we be able to eventually replace this with the bootstrap
program in https://github.com/kubernetes/test-infra/tree/master/bootstrap?
"""
import logging
import subprocess
import os
import shutil

# Default name for the repo organization and name.
# This should match the values used in Go imports.
GO_REPO_OWNER = "jlewi"
GO_REPO_NAME = "mlkube.io"


def run(command, cwd=None):
  """Run the command as a subprocess"""
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
  run(["git", "clone", repo, dest])

  # If this is a presubmit PULL_PULL_SHA will be set see:
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

def main():
  src_dir, sha = clone_repo()

  # Execute the runner.
  runner = os.path.join(src_dir, "test-infra", "runner.py")
  run(["python", runner, "--src_dir=" + src_dir, "--sha=" + sha])

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  main()
