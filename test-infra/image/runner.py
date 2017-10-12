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

import logging
import subprocess
import os
import shutil

def run(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  subprocess.check_call(command, cwd=cwd)

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)

  go_path = os.getenv("GOPATH")
  repo_owner = os.getenv("REPO_OWNER")
  repo_name = os.getenv("REPO_NAME")

  # Clone mlkube
  repo = "https://github.com/{0}/{1}.git".format(repo_owner, repo_name)
  logging.info("repo %s", repo)

  src_dir = os.path.join(go_path, "src/github.com/jlewi")
  os.makedirs(src_dir)
  dest = os.path.join(src_dir, "mlkube.io")

  # TODO(jlewi): How can we figure out what branch
  run(["git", "clone",  repo, dest])

  pull_sha = os.getenv('PULL_PULL_SHA')
  if pull_sha:
    run(["git", "checkout", pull_sha], cwd=dest)

  # Install dependencies
  run(["glide", "install"], cwd=dest)

  # We need to remove the vendored dependencies of apiextensions otherwise we
  # get type conflicts.
  shutil.rmtree(os.path.join(dest,
                             "vendor/k8s.io/apiextensions-apiserver/vendor"))

  # Build and push the image
  run(["./images/tf_operator/build_and_push.py",
       "--registry=gcr.io/mlkube-testing"], cwd=dest)
