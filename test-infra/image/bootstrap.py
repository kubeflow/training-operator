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
from __future__ import print_function

import json
import logging
import os
import shutil
import subprocess

# Default name for the repo organization and name.
# This should match the values used in Go imports.
GO_REPO_OWNER = "tensorflow"
GO_REPO_NAME = "k8s"


def run(command, cwd=None, env=None):
  logging.info("Running: %s", " ".join(command))
  if not env:
    env = os.environ
  subprocess.check_call(command, cwd=cwd, env=env)


def run_and_output(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  # The output won't be available until the command completes.
  # So prefer using run if we don't need to return the output.
  output = subprocess.check_output(command, cwd=cwd).decode("utf-8")
  print(output)
  return output


def clone_repo():
  """Clone the repo.

  Returns:
    src_path: This is the root path for the training code.
    sha: The sha number of the repo.
  """
  go_path = os.getenv("GOPATH")
  # REPO_OWNER and REPO_NAME are the environment variables set by Prow.
  # REPO_OWNER and REPO_NAME won't be set for periodic jobs so we resort
  # to default values.
  repo_owner = os.getenv("REPO_OWNER", GO_REPO_OWNER)
  repo_name = os.getenv("REPO_NAME", GO_REPO_NAME)

  if bool(repo_owner) != bool(repo_name):
    raise ValueError("Either set both environment variables "
                     "REPO_OWNER and REPO_NAME or set neither.")

  src_dir = os.path.join(go_path, "src/github.com", GO_REPO_OWNER)
  dest = os.path.join(src_dir, GO_REPO_NAME)

  # Clone mlkube
  repo = "https://github.com/{0}/{1}.git".format(repo_owner, repo_name)
  logging.info("repo %s", repo)

  os.makedirs(src_dir)

  # TODO(jlewi): How can we figure out what branch
  run(["git", "clone", repo, dest])

  # If this is a presubmit PULL_PULL_SHA will be set see:
  # https://github.com/kubernetes/test-infra/tree/master/prow#job-evironment-variables
  sha = ""
  pull_number = os.getenv("PULL_NUMBER", "")

  if pull_number:
    sha = os.getenv("PULL_PULL_SHA", "")
    # Its a presubmit job sob since we are testing a Pull request.
    run(["git", "fetch", "origin",
         "pull/{0}/head:pr".format(pull_number)],
        cwd=dest)
  else:
    # For postsubmits PULL_BASE_SHA will be set
    sha = os.getenv("PULL_BASE_SHA", "")

  if sha:
    run(["git", "checkout", sha], cwd=dest)

  # Get the actual git hash.
  # This ensures even for periodic jobs which don't set the sha we know
  # the version of the code tested.
  sha = run_and_output(["git", "rev-parse", "HEAD"], cwd=dest)

  # Install dependencies
  # TODO(jlewi): We should probably move this into runner.py so that
  # output is captured in build-log.txt
  run(["glide", "install"], cwd=dest)

  # We need to remove the vendored dependencies of apiextensions otherwise we
  # get type conflicts.
  shutil.rmtree(os.path.join(dest,
                             "vendor/k8s.io/apiextensions-apiserver/vendor"))

  return dest, sha


def main():
  this_dir = os.path.dirname(__file__)
  version_file = os.path.join(this_dir, "version.json")
  if os.path.exists(version_file):
    # Print out version information so we know what container we ran in.
    with open(version_file) as hf:
      version = json.load(hf)
      logging.info("Image info:\n%s", json.dumps(version, indent=2,
                                                 sort_keys=True))
  else:
    logging.warn("Could not find file: %s", version_file)

  # Print environment variables.
  # This is useful for debugging and understanding the information set by prow.
  names = os.environ.keys()
  names.sort()
  logging.info("Environment Variables")
  for n in names:
    logging.info("%s=%s", n, os.environ[n])
  logging.info("End Environment Variables")

  src_dir, sha = clone_repo()

  # Execute the runner.
  runner = os.path.join(src_dir, "test-infra", "runner.py")
  # Ensure src_dir ends up on the path.
  new_env = os.environ.copy()
  py_path = new_env.get("PYTHONPATH", "")
  if py_path:
    py_path += ":"
  py_path += src_dir
  new_env["PYTHONPATH"] = py_path
  run(["python", runner, "--src_dir=" + src_dir, "--sha=" + sha], cwd=src_dir,
      env=new_env)


if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  main()
