#!/usr/bin/python
from __future__ import print_function

import argparse
import datetime
import hashlib
import logging
import os
import shutil
import subprocess
import sys
import tempfile

import yaml

# TODO(jlewi): build_and_push.py should be obsolete. We should be able to
# use py/release.py

def GetGitHash(root_dir):
  # The image tag is based on the githash.
  git_hash = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"],
                                     cwd=root_dir).decode("utf-8")
  git_hash = git_hash.strip()

  modified_files = subprocess.check_output(["git", "ls-files", "--modified"],
                                           cwd=root_dir)
  untracked_files = subprocess.check_output(
      ["git", "ls-files", "--others", "--exclude-standard"], cwd=root_dir)
  if modified_files or untracked_files:
    diff = subprocess.check_output(["git", "diff"], cwd=root_dir)

    sha = hashlib.sha256()
    sha.update(diff)
    diffhash = sha.hexdigest()[0:7]
    git_hash = "{0}-dirty-{1}".format(git_hash, diffhash)

  return git_hash


def run(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  subprocess.check_call(command, cwd=cwd)


def run_and_output(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  # The output won't be available until the command completes.
  # So prefer using run if we don't need to return the output.
  output = subprocess.check_output(command, cwd=cwd).decode("utf-8")
  print(output)
  return output


def main():  # pylint: disable=too-many-locals, too-many-statements
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals, too-many-statements
  parser = argparse.ArgumentParser(
      description="Build docker image for TFJob CRD.")

  # TODO(jlewi) We should make registry required to avoid people accidentally
  # pushing to tf-on-k8s-dogfood by default.
  parser.add_argument(
      "--registry",
      default="gcr.io/tf-on-k8s-dogfood",
      type=str,
      help="The docker registry to use.")

  parser.add_argument(
      "--project",
      default="",
      type=str,
      help="Project to use with Google Container Builder when using GCB.")

  parser.add_argument(
      "--output",
      default="",
      type=str,
      help="Path to write a YAML file with build info.")

  parser.add_argument("--gcb", dest="use_gcb", action="store_true",
                      help="Use Google Container Builder to build the image.")
  parser.add_argument("--no-gcb", dest="use_gcb", action="store_false",
                      help="Use Docker to build the image.")
  parser.add_argument("--no-push", dest="should_push", action="store_false",
                      help="Do not push the image once build is finished.")

  parser.set_defaults(use_gcb=False)

  args = parser.parse_args()

  if args.use_gcb and not args.project:
    logging.fatal("--project must be set when using Google Container Builder")
    sys.exit(-1)

  this_file = __file__
  images_dir = os.path.dirname(this_file)
  root_dir = os.path.abspath(os.path.join(images_dir, os.pardir, os.pardir))

  context_dir = tempfile.mkdtemp(prefix="tmpTFJobCrdContext")
  logging.info("context_dir: %s", context_dir)
  if not os.path.exists(context_dir):
    os.makedirs(context_dir)

  # Build the go binaries
  go_path = os.environ["GOPATH"]

  targets = [
      "github.com/tensorflow/k8s/cmd/tf_operator",
      "github.com/tensorflow/k8s/test/e2e",
      "github.com/tensorflow/k8s/dashboard/backend",
  ]
  for t in targets:
    subprocess.check_call(["go", "install", t])

  # Resolving dashboard's front-end dependencies
  subprocess.check_call(["yarn", "--cwd", "./dashboard/frontend", "install"])
  # Building dashboard's front-end
  subprocess.check_call(["yarn", "--cwd", "./dashboard/frontend", "build"])

  root_dir = os.path.abspath(os.path.join(images_dir, '..', '..'))
  # List of paths to copy relative to root.
  sources = [
      "build/images/tf_operator/Dockerfile",
      os.path.join(go_path, "bin/tf_operator"),
      os.path.join(go_path, "bin/e2e"),
      os.path.join(go_path, "bin/backend"),
      "dashboard/frontend/build"
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

  n = datetime.datetime.now()
  image = image_base + ":" + n.strftime("v%Y%m%d") + "-" + GetGitHash(root_dir)
  latest_image = image_base + ":latest"

  if args.use_gcb:
    run(["gcloud", "container", "builds", "submit", context_dir,
         "--tag=" + image, "--project=" + args.project])

    # Add the latest tag.
    run(["gcloud", "container", "images", "add-tag", "--quiet", image,
         latest_image])

  else:
    run(["docker", "build", "-t", image, context_dir])
    logging.info("Built image: %s", image)

    run(["docker", "tag", image, latest_image])

    if args.should_push:
      run(["gcloud", "docker", "--", "push", image])
      logging.info("Pushed image: %s", image)

      run(["gcloud", "docker", "--", "push", latest_image])
      logging.info("Pushed image: %s", latest_image)

  if args.output:
    logging.info("Writing build information to %s", args.output)
    output = {"image": image}
    with open(args.output, mode='w') as hf:
      yaml.dump(output, hf)

if __name__ == "__main__":
  main()
