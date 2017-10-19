#!/usr/bin/python
import argparse
import datetime
import hashlib
import logging
import os
import re
import shutil
import subprocess
import sys
import tempfile
import yaml

def GetGitHash():
  # The image tag is based on the githash.
  git_hash = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"]).decode("utf-8") 
  git_hash = git_hash.strip()
  
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

def run(command, cwd=None):
  logging.info("Running: %s", " ".join(command))
  subprocess.check_call(command, cwd=cwd)

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  parser = argparse.ArgumentParser(
      description="Build docker image for TfJob CRD.")

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

  context_dir = tempfile.mkdtemp(prefix="tmpTfJobCrdContext")
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
      "grpc_tensorflow_server/grpc_tensorflow_server.py"
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
  image = image_base + ":" + n.strftime("v%Y%m%d") + "-" + GetGitHash()
  if args.use_gcb:
    run(["gcloud", "container", "builds", "submit", context_dir,
         "--tag=" + image, "--project=" + args.project ])
  else:
    run(["docker", "build", "-t", image,  context_dir])
    logging.info("Built image: %s", image)  
    
    if args.should_push:
      run(["gcloud", "docker", "--", "push", image])
      logging.info("Pushed image: %s", image)

  if args.output:
    logging.info("Writing build information to %s", args.output)
    output = { "image": image }
    with open(args.output, mode='w') as hf:
      yaml.dump(output, hf)
