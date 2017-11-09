#!/usr/bin/python
"""Release a new Docker image and helm package.

This script should be run from the root directory of the repo.
"""

import argparse
import datetime
import glob
import json
import logging
import os
import shutil
import sys
import tempfile
import time

import yaml
from google.cloud import storage  # pylint: disable=no-name-in-module

from py import build_and_push_image
from py import util

REPO_ORG = "tensorflow"
REPO_NAME = "k8s"

RESULTS_BUCKET = "mlkube-testing-results"
JOB_NAME = "tf-k8s-postsubmit"

GCB_PROJECT = "tf-on-k8s-releasing"


def get_latest_green_presubmit(gcs_client):
  bucket = gcs_client.get_bucket(RESULTS_BUCKET)
  blob = bucket.blob(os.path.join(JOB_NAME, "latest_green.json"))
  contents = blob.download_as_string()

  results = json.loads(contents)

  if results.get("status", "").lower() != "passing":
    raise ValueError("latest results aren't green.")

  return results.get("sha", "").strip()


def update_values(values_file, image):
  """Update the values file for the helm package to use the new image."""

  # We want to preserve comments so we don't use the yaml library.
  with open(values_file) as hf:
    lines = hf.readlines()

  with open(values_file, "w") as hf:
    for l in lines:
      if l.startswith("image:"):
        hf.write("image: {0}\n".format(image))
      else:
        hf.write(l)


def update_chart(chart_file, version):
  """Append the version number to the version number in chart.yaml"""
  with open(chart_file) as hf:
    info = yaml.load(hf)
  info["version"] += "-" + version
  info["appVersion"] += "-" + version

  with open(chart_file, "w") as hf:
    yaml.dump(info, hf)


def get_last_release(bucket):
  """Return the sha of the last release.

  Args:
    bucket: A google cloud storage bucket object

  Returns:
    sha: The sha of the latest release.
  """

  path = "latest_release.json"

  blob = bucket.blob(path)

  if not blob.exists():
    logging.info("File %s doesn't exist.", util.to_gcs_uri(bucket.name, path))
    return ""

  contents = blob.download_as_string()

  data = json.loads(contents)
  return data.get("sha", "").strip()

def create_latest(bucket, sha, target):
  """Create a file in GCS with information about the latest release.

  Args:
    bucket: A google cloud storage bucket object
    sha: SHA of the release we just created
    target: The GCS path of the release we just produced.
  """
  path = os.path.join("latest_release.json")

  logging.info("Creating GCS output: %s", util.to_gcs_uri(bucket.name, path))

  data = {
      "sha": sha.strip(),
      "target": target,
  }
  blob = bucket.blob(path)
  blob.upload_from_string(json.dumps(data))


def build_operator_image(root_dir, registry, output_path=None, project=None,
                         should_push=True):
  """Build the main docker image for the TfJob CRD.
  Args:
    root_dir: Root directory of the repository.
    registry: The registry to use.
    output_path: Path to write build information for.
    project: If set it will be built using GCB.
  """
  context_dir = tempfile.mkdtemp(prefix="tmpTfJobCrdContext")
  logging.info("context_dir: %s", context_dir)
  if not os.path.exists(context_dir):
    os.makedirs(context_dir)

  # Build the go binaries
  go_path = os.environ["GOPATH"]

  targets = [
      "github.com/tensorflow/k8s/cmd/tf_operator",
      "github.com/tensorflow/k8s/test/e2e",
  ]
  for t in targets:
    util.run(["go", "install", t])

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

  image_base = registry + "/tf_operator"

  n = datetime.datetime.now()
  image = (image_base + ":" + n.strftime("v%Y%m%d") + "-" +
           build_and_push_image.GetGitHash(root_dir))
  latest_image = image_base + ":latest"

  if project:
    util.run(["gcloud", "container", "builds", "submit", context_dir,
              "--tag=" + image, "--project=" + project])

    # Add the latest tag.
    util.run(["gcloud", "container", "images", "add-tag", "--quiet", image,
              latest_image])

  else:
    util.run(["docker", "build", "-t", image, context_dir])
    logging.info("Built image: %s", image)

    util.run(["docker", "tag", image, latest_image])

    if should_push:
      util.run(["gcloud", "docker", "--", "push", image])
      logging.info("Pushed image: %s", image)

      util.run(["gcloud", "docker", "--", "push", latest_image])
      logging.info("Pushed image: %s", latest_image)

  if output_path:
    logging.info("Writing build information to %s", output_path)
    output = {"image": image}
    with open(output_path, mode='w') as hf:
      yaml.dump(output, hf)

def build_and_push_artifacts(go_dir, src_dir, registry, publish_path=None,
                             gcb_project=None):
  """Build and push the artifacts.

  Args:
    go_dir: The GOPATH directory
    src_dir: The root directory where we checked out the repo.
    registry: Docker registry to use.
    publish_path: (Optional) The GCS path where artifacts should be published.
       Set to none to only build locally.
    gcb_project: The project to use with GCB to build docker images.
      If set to none uses docker to build.
  """
  # Update the GOPATH to the temporary directory.
  env = os.environ.copy()
  if go_dir:
    env["GOPATH"] = go_dir

  bin_dir = os.path.join(src_dir, "bin")
  if not os.path.exists(bin_dir):
    os.makedirs(bin_dir)

  build_info_file = os.path.join(bin_dir, "build_info.yaml")

  build_operator_image(src_dir, registry, project=gcb_project,
                       output_path=build_info_file)

  with open(build_info_file) as hf:
    build_info = yaml.load(hf)

  version = build_info["image"].split(":")[-1]
  values_file = os.path.join(src_dir, "tf-job-operator-chart", "values.yaml")
  update_values(values_file, build_info["image"])

  chart_file = os.path.join(src_dir, "tf-job-operator-chart", "Chart.yaml")
  update_chart(chart_file, version)

  # Delete any existing matches because we assume there is only 1 below.
  matches = glob.glob(os.path.join(bin_dir, "tf-job-operator-chart*.tgz"))
  for m in matches:
    logging.info("Delete previous build: %s", m)
    os.unlink(m)

  util.run(["helm", "package", "--destination=" + bin_dir,
            "./tf-job-operator-chart"], cwd=src_dir)

  matches = glob.glob(os.path.join(bin_dir, "tf-job-operator-chart*.tgz"))

  if len(matches) != 1:
    raise ValueError(
      "Expected 1 chart archive to match but found {0}".format(matches))

  chart_archive = matches[0]

  release_path = version

  targets = [
    os.path.join(release_path, os.path.basename(chart_archive)),
    "latest/tf-job-operator-chart-latest.tgz",
  ]

  if publish_path:
    bucket_name, base_path = util.split_gcs_uri(publish_path)
    for t in targets:
      blob = bucket.blob(os.path.join(base_path, t))
      gcs_path = util.to_gcs_uri(bucket_name, t)
      if blob.exists() and not t.startswith("latest"):
        logging.warn("%s already exists", gcs_path)
        continue
      logging.info("Uploading %s to %s.", chart_archive, gcs_path)
      blob.upload_from_filename(chart_archive)

    create_latest(bucket, sha, util.to_gcs_uri(bucket_name, targets[0]))

def build_local(args):
  """Build the artifacts from the local copy of the code."""
  go_dir = None
  src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
  build_and_push_artifacts(go_dir, src_dir, args.registry)

def build_once(releases_path):  # pylint: disable=too-many-locals
  """Build the artifacts once

  Args:
    releases_path: GCS prefix where releases should be published.
  """
  gcs_client = storage.Client()
  sha = get_latest_green_presubmit(gcs_client)

  bucket_name, path = util.split_gcs_uri(releases_path)
  bucket = gcs_client.get_bucket(bucket_name)

  logging.info("Latest passing postsubmit is %s", sha)

  last_release_sha = get_last_release(bucket_name)
  logging.info("Most recent release was for %s", last_release_sha)

  if sha == last_release_sha:
    logging.info("Already cut release for %s", sha)
    return

  go_dir = tempfile.mkdtemp(prefix="tmpTfJobSrc")
  logging.info("Temporary go_dir: %s", go_dir)

  src_dir = os.path.join(go_dir, "src", "github.com", REPO_ORG, REPO_NAME)

  _, sha = util.clone_repo(src_dir, util.MASTER_REPO_OWNER,
                           util.MASTER_REPO_NAME, sha)

  build_and_push_artifacts(go_dir, src_dir, bucket_name)

def main():  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO) # pylint: disable=too-many-locals
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

  # create the top-level parser
  parser = argparse.ArgumentParser(
      description="Build the release artifacts.")
  subparsers = parser.add_subparsers()

  # Create the parser for the "local" mode.
  # This mode builds the artifacts from the local copy of the code.
  parser_local = subparsers.add_parser(
    "local",
    help="Build the artifacts from the local copy of the code.")
  parser.add_argument(
    "--registry",
      default="gcr.io/mlkube-testing",
      type=str,
      help="The docker registry to use.")
  #parser_local.add_argument('y', type=float)
  parser_local.set_defaults(func=build_local)

  #>>> # create the parser for the "bar" command
  #>>> parser_bar = subparsers.add_parser('bar')
  #>>> parser_bar.add_argument('z')
  #>>> parser_bar.set_defaults(func=bar)
  #>>>
  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)

  #parser.add_argument(
      #"--releases_path",
      #default=None,
      #required=True,
      #type=str,
      #help="The GCS location where artifacts should be pushed.")

  #parser.add_argument(
    #"--check_interval_secs",
      #default=0,
      #type=int,
      #help=("How often to periodically check to see if there is a new passing "
            #"postsubmit. If set to 0 (default) script will run once and exit."))

  # TODO(jlewi): Should pass along unknown arguments to build and push.
  #args, _ = parser.parse_known_args()

  #while True:
    #logging.info("Checking latest postsubmit results")
    #build_once(args.releases_bucket)

    #if args.check_interval_secs > 0:
      #logging.info("Sleep %s seconds before checking for a postsubmit.",
                   #args.check_interval_secs)
      #time.sleep(args.check_interval_secs)
    #else:
      #break

if __name__ == "__main__":
  main()
