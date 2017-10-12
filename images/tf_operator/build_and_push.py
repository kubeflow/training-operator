#!/usr/bin/python
import argparse
import hashlib
import jinja2
import logging
import os
import re
import shutil
import subprocess
import tempfile


def GetGitHash():
  # The image tag is based on the githash.
  git_hash = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"])
  git_hash=git_hash.strip()
  modified_files = subprocess.check_output(["git", "ls-files", "--modified"])
  untracked_files = subprocess.check_output(
      ["git", "ls-files", "--others", "--exclude-standard"])
  if modified_files or untracked_files:
    diff= subprocess.check_output(["git", "diff"])
    sha = hashlib.sha256()
    sha.update(diff)
    diffhash = sha.hexdigest()[0:7]
    git_hash = "{0}-dirty-{1}".format(git_hash, diffhash)
  return git_hash

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  parser = argparse.ArgumentParser(
      description="Build docker image for TfJob CRD.")

  parser.add_argument(
      "--registry",
      default="gcr.io/tf-on-k8s-dogfood",
      type=str,
      help="The docker registry to use.")

  args = parser.parse_args()


  this_file = __file__
  images_dir = os.path.dirname(this_file)

  context_dir = tempfile.mkdtemp(prefix="tmpTfJobCrdContentxt")
  logging.info("context_dir: %s", context_dir)
  if not os.path.exists(context_dir):
    os.makedirs(context_dir)
  dockerfile = os.path.join(context_dir, 'Dockerfile')

  # Build the go binaries
  go_path = os.environ["GOPATH"]

  targets = [
      "github.com/jlewi/mlkube.io/cmd/tf_operator",
      "github.com/jlewi/mlkube.io/test/e2e",
    ]
  for t in targets:
    subprocess.check_call(["go", "install", t])

  root_dir = os.path.abspath(os.path.join(images_dir, '..', '..'))
  # List of paths to copy relative to root.
  sources = [
      "images/tf_operator/Dockerfile",
      os.path.join(go_path, "bin/tf_operator"),
      os.path.join(go_path, "bin/e2e"),
    ]

  for s in sources:
    src_path = os.path.join(root_dir, s)
    dest_path = os.path.join(context_dir, os.path.basename(s))
    if os.path.exists(dest_path):
      os.unlink(dest_path)
    if os.path.isdir(src_path):
      shutil.copytree(src_path, dest_path)
    else:
      shutil.copyfile(src_path, dest_path)

  image_base = args.registry + "/tf_operator"

  image = image_base + ":" + GetGitHash()
  subprocess.check_call(["docker", "build", "-t", image,  context_dir])
  logging.info("Built image: %s", image)
  subprocess.check_call(["gcloud", "docker", "--", "push", image])
  logging.info("Pushed image: %s", image)
