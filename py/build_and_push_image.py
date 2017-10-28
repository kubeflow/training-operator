#!/usr/bin/python
#
# This is a helper script for building Docker images.
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
  untracked_files = subprocess.check_output(["git", "ls-files", "--others",
                                             "--exclude-standard"])
  if modified_files or untracked_files:
    diff= subprocess.check_output(["git", "diff"])
    sha = hashlib.sha256()
    sha.update(diff)
    diffhash = sha.hexdigest()[0:7]
    git_hash = "{0}-dirty-{1}".format(git_hash, diffhash)
  return git_hash

def run_and_stream(cmd):
  logging.info("Running %s", " ".join(cmd))
  process = subprocess.Popen(cmd, stdout=subprocess.PIPE,
                             stderr=subprocess.STDOUT)

  while process.poll() is None:
    process.stdout.flush()
    for line in iter(process.stdout.readline, ''):
      logging.info(line.strip())

  process.stdout.flush()
  for line in iter(process.stdout.readline, ''):
    logging.info(line.strip())

  if process.returncode != 0:
    raise ValueError("cmd: {0} exited with code {1}".format(
      " ".join(cmd), process.returncode))

def build_and_push(dockerfile_template, image, modes=["cpu", "gpu"],
                   skip_push=False, base_images=None):
  loader = jinja2.FileSystemLoader(os.path.dirname(dockerfile_template))

  if not base_images:
    raise ValueError("base_images must be provided.")

  images = {}
  for mode in modes:
    dockerfile_contents = jinja2.Environment(loader=loader).get_template(
      os.path.basename(dockerfile_template)).render(base_image=base_images[mode])
    context_dir = tempfile.mkdtemp(prefix="tmpTfJobSampleContentxt")
    logging.info("context_dir: %s", context_dir)
    shutil.rmtree(context_dir)
    shutil.copytree(os.path.dirname(dockerfile_template), context_dir)
    dockerfile = os.path.join(context_dir, 'Dockerfile')
    with open(dockerfile, 'w') as hf:
      hf.write(dockerfile_contents)

    full_image = image + "-" + mode

    full_image += ":" + GetGitHash()
    run_and_stream(["docker", "build", "-t", full_image,  context_dir])
    logging.info("Built image: %s", full_image)

    images[mode] = full_image
    if not skip_push:
      if "gcr.io" in full_image:
        run_and_stream(["gcloud", "docker", "--", "push", full_image])
      else:
        run_and_stream(["docker", "--", "push", full_image])
      logging.info("Pushed image: %s", full_image)
  return images

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  parser = argparse.ArgumentParser(
      description="Build Docker images based off of TensorFlow.")

  parser.add_argument(
      "--image",
      default="gcr.io/tf-on-k8s-dogfood",
      type=str,
      help="The image path to use; mode will be applied as a suffix.")

  parser.add_argument(
      "--dockerfile",
      required=True,
      type=str,
      help="The path to the Dockerfile")

  # TODO(jlewi): Should we make this a list so we can build both images with one command.
  parser.add_argument(
      '--mode',
        default=["cpu", "gpu"],
        dest = "modes",
        action = "append",
        help='Which image to build; options are cpu or gpu')


  parser.add_argument("--no-push", dest="should_push", action="store_false",
                        help="Do not push the image once build is finished.")

  args = parser.parse_args()

  base_images = {
    "cpu": "gcr.io/tensorflow/tensorflow:1.3.0",
    "gpu": "gcr.io/tensorflow/tensorflow:1.3.0-gpu",
  }

  build_and_push(args.dockerfile, args.modes, not args.should_push, base_images)
