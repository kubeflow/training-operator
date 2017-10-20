#!/usr/bin/python
"""Release a new Docker image and helm package."""

from google.cloud import storage
import json
import tempfile
from py import util

REPO_ORG = "jlewi"
REPO_NAME = "mlkube.io"

RESULTS_BUCKET = "mlkube-testing-results"
JOB_NAME = "mlkube-postsubmit-build"

def get_latest_green_presubmit():
  bucket = gcs_client.get_bucket(bucket)
  latest_results = os.path.join(JOB_NAME)
  blob = bucket.blob(os.path.join(JOB_NAME, "latest_green.json"))
  contents = blob.download_as_string()

  results = json.loads(contents)

  if results.get("status", "").lower() != "passing":
    return ValueError("latest results aren't green."))

  return results.get("sha", "")

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  parser = argparse.ArgumentParser(
      description="Release artifacts for TfJob.")

  parser.add_argument(
      "--registry",
      default="gcr.io/tf-on-k8s-dogfood",
      type=str,
      help="The docker registry to use.")

  _, unknwon_args = parser.parse

  if args.use_gcb and not args.project:
    logging.fatal("--project must be set when using Google Container Builder")
    sys.exit(-1)

  this_file = __file__
  images_dir = os.path.dirname(this_file)
  root_dir = os.path.abspath(os.path.join(images_dir, os.pardir, os.pardir))

  gcs_client = storage.Client()
  sha = get_latest_green_presubmit(gcs_client)

  src_dir = tempfile.mkdtemp(prefix="tmpTfJobSrc")
  logging.info("src_dir: %s", src_dir)



