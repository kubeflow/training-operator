#!/usr/bin/python
"""Release a new Docker image and helm package.

This script should be run from the root directory of the repo.
"""

import argparse
import glob
import json
import logging
import os
import tempfile
import time

import yaml
from google.cloud import storage  # pylint: disable=no-name-in-module

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

def build_once(bucket_name):  # pylint: disable=too-many-locals
  gcs_client = storage.Client()
  sha = get_latest_green_presubmit(gcs_client)

  bucket = gcs_client.get_bucket(bucket_name)

  logging.info("Latest passing postsubmit is %s", sha)

  last_release_sha = get_last_release(bucket)
  logging.info("Most recent release was for %s", last_release_sha)

  if sha == last_release_sha:
    logging.info("Already cut release for %s", sha)
    return

  go_dir = tempfile.mkdtemp(prefix="tmpTfJobSrc")
  logging.info("Temporary go_dir: %s", go_dir)

  src_dir = os.path.join(go_dir, "src", "github.com", REPO_ORG, REPO_NAME)

  _, sha = util.clone_repo(src_dir, util.MASTER_REPO_OWNER,
                           util.MASTER_REPO_NAME, sha)

  # Update the GOPATH to the temporary directory.
  env = os.environ.copy()
  env["GOPATH"] = go_dir
  build_info_file = os.path.join(src_dir, "build_info.yaml")
  util.run([os.path.join(src_dir, "images", "tf_operator", "build_and_push.py"),
            "--gcb", "--project=" + GCB_PROJECT,
            "--output=" + build_info_file], cwd=src_dir, env=env)

  with open(build_info_file) as hf:
    build_info = yaml.load(hf)

  version = build_info["image"].split(":")[-1]
  values_file = os.path.join(src_dir, "tf-job-operator-chart", "values.yaml")
  update_values(values_file, build_info["image"])

  chart_file = os.path.join(src_dir, "tf-job-operator-chart", "Chart.yaml")
  update_chart(chart_file, version)

  util.run(["helm", "package", "./tf-job-operator-chart"], cwd=src_dir)

  matches = glob.glob(os.path.join(src_dir, "tf-job-operator-chart*.tgz"))

  if len(matches) != 1:
    raise ValueError(
        "Expected 1 chart archive to match but found {0}".format(matches))

  chart_archive = matches[0]

  release_path = version

  targets = [
      os.path.join(release_path, os.path.basename(chart_archive)),
      "latest/tf-job-operator-chart-latest.tgz",
    ]

  for t in targets:
    blob = bucket.blob(t)
    gcs_path = util.to_gcs_uri(bucket_name, t)
    if blob.exists() and not t.startswith("latest"):
      logging.warn("%s already exists", gcs_path)
      continue
    logging.info("Uploading %s to %s.", chart_archive, gcs_path)
    blob.upload_from_filename(chart_archive)

  create_latest(bucket, sha, util.to_gcs_uri(bucket_name, targets[0]))

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

  parser = argparse.ArgumentParser(
      description="Release artifacts for TfJob.")

  parser.add_argument(
      "--releases_bucket",
      default="tf-on-k8s-dogfood-releases",
      type=str,
      help="The bucket to publish releases to.")

  parser.add_argument(
    "--check_interval_secs",
      default=0,
      type=int,
      help=("How often to periodically check to see if there is a new passing "
            "postsubmit. If set to 0 (default) script will run once and exit."))

  # TODO(jlewi): Should pass along unknown arguments to build and push.
  args, _ = parser.parse_known_args()

  while True:
    logging.info("Checking latest postsubmit results")
    build_once(args.releases_bucket)

    if args.check_interval_secs > 0:
      logging.info("Sleep %s seconds before checking for a postsubmit.",
                   args.check_interval_secs)
      time.sleep(args.check_interval_secs)
    else:
      break

if __name__ == "__main__":
  main()
