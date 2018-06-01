#!/usr/bin/python
"""Build a new Docker image and helm package.

This module assumes py is a top level python package.
"""
# TODO(jlewi): After we migrate to using Argo for our tests and releases_path
# I think we should be able to get rid of most of the clone functions because
# a separate step in the workflow is responsible for checking out the code
# and it doesn't use this script.
import argparse
import datetime
import json
import logging
import os
import shutil
import tempfile
import platform
import yaml
from google.cloud import storage  # pylint: disable=no-name-in-module

from py import build_and_push_image
from py import util

# Repo org and name can be set via environment variables when running
# on PROW. But we choose sensible defaults so that we can run locally without
# setting defaults.
REPO_ORG = os.getenv("REPO_OWNER", "kubeflow")
REPO_NAME = os.getenv("REPO_NAME", "tf-operator")

RESULTS_BUCKET = "kubeflow-ci-results"
JOB_NAME = "tf-k8s-postsubmit"

GCB_PROJECT = "tf-on-k8s-releasing"

# Directory to checkout the source.
REPO_DIR = "git_tensorflow_k8s"


def get_latest_green_presubmit(gcs_client):
  """Find the commit corresponding to the latest passing postsubmit."""
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


def build_operator_image(root_dir,
                         registry,
                         project=None,
                         should_push=True,
                         version_tag=None):
  """Build the main docker image for the TFJob CRD.
  Args:
    root_dir: Root directory of the repository.
    registry: The registry to use.
    project: If set it will be built using GCB.
    should_push: Should push the image to the registry, Defaule is True.
    version_tag: Optional tag for the version. If not specified derive
      the tag from the git hash.
  Returns:
    build_info: Dictionary containing information about the build.
  """
  context_dir = tempfile.mkdtemp(prefix="tmpTFJobCrdContext")
  logging.info("context_dir: %s", context_dir)
  if not os.path.exists(context_dir):
    os.makedirs(context_dir)

  # Build the go binaries
  go_path = os.environ["GOPATH"]
  commit = build_and_push_image.GetGitHash(root_dir)

  targets = [
    "github.com/kubeflow/tf-operator/cmd/tf-operator",
    "github.com/kubeflow/tf-operator/cmd/tf-operator.v2",
    "github.com/kubeflow/tf-operator/test/e2e",
    "github.com/kubeflow/tf-operator/dashboard/backend",
  ]
  for t in targets:
    if t in ["github.com/kubeflow/tf-operator/cmd/tf-operator",
             "github.com/kubeflow/tf-operator/cmd/tf-operator.v2"]:
      util.run([
        "go", "install", "-ldflags",
        "-X github.com/kubeflow/tf-operator/pkg/version.GitSHA={}".format(commit), t
      ])
    util.run(["go", "install", t])

  # Dashboard's frontend:
  # Resolving dashboard's front-end dependencies
  util.run(
    ["yarn", "--cwd", "{}/dashboard/frontend".format(root_dir), "install"])
  # Building dashboard's front-end
  util.run(["yarn", "--cwd", "{}/dashboard/frontend".format(root_dir), "build"])

  # If the release is not done from a Linux machine
  # we need to grab the artefacts from /bin/linux_amd64
  bin_path = "bin"
  if platform.system() != "Linux":
    bin_path += "/linux_amd64"

  # List of paths to copy relative to root.
  sources = [
    "build/images/tf_operator/Dockerfile",
    "examples/tf_sample/tf_sample/tf_smoke.py",
    os.path.join(go_path, bin_path, "tf-operator"),
    os.path.join(go_path, bin_path, "tf-operator.v2"),
    os.path.join(go_path, bin_path, "e2e"),
    os.path.join(go_path, bin_path, "backend"), "dashboard/frontend/build"
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

  if not version_tag:
    logging.info("No version tag specified; computing tag automatically.")
    n = datetime.datetime.now()
    version_tag = n.strftime("v%Y%m%d") + "-" + commit
  logging.info("Using version tag: %s", version_tag)
  image = image_base + ":" + version_tag
  latest_image = image_base + ":latest"

  if project:
    util.run([
      "gcloud", "container", "builds", "submit", context_dir, "--tag=" + image,
      "--project=" + project
    ])

    # Add the latest tag.
    util.run([
      "gcloud", "container", "images", "add-tag", "--quiet", image, latest_image
    ])

  else:
    util.run(["docker", "build", "-t", image, context_dir])
    logging.info("Built image: %s", image)

    util.run(["docker", "tag", image, latest_image])

    if should_push:
      _push_image(image, latest_image)

  output = {
    "image": image,
    "commit": commit,
  }
  return output


def _push_image(image, latest_image):
  if "gcr.io" in image:
    util.run(["gcloud", "docker", "--", "push", image])
    logging.info("Pushed image: %s", image)

    util.run(["gcloud", "docker", "--", "push", latest_image])
    logging.info("Pushed image: %s", latest_image)

  else:
    util.run(["docker", "push", image])
    logging.info("Pushed image: %s", image)

    util.run(["docker", "push", latest_image])
    logging.info("Pushed image: %s", latest_image)


def build_and_push_artifacts(go_dir,
                             src_dir,
                             registry,
                             gcb_project=None,
                             build_info_path=None,
                             version_tag=None):
  """Build and push the artifacts.

  Args:
    go_dir: The GOPATH directory
    src_dir: The root directory where we checked out the repo.
    registry: Docker registry to use.
    gcb_project: The project to use with GCB to build docker images.
      If set to none uses docker to build.
    build_info_path: (Optional): GCS location to write YAML file containing
      information about the build.
    version_tag: (Optional): The tag to use for the image.
  """
  # Update the GOPATH to the temporary directory.
  env = os.environ.copy()
  if go_dir:
    env["GOPATH"] = go_dir

  bin_dir = os.path.join(src_dir, "bin")
  if not os.path.exists(bin_dir):
    os.makedirs(bin_dir)

  build_info = build_operator_image(
    src_dir, registry, project=gcb_project, version_tag=version_tag)

  # Always write to the bin dir.
  paths = [os.path.join(bin_dir, "build_info.yaml")]

  if build_info_path:
    paths.append(build_info_path)

  write_build_info(build_info, paths, project=gcb_project)


def write_build_info(build_info, paths, project=None):
  """Write the build info files.
  """
  gcs_client = None

  contents = yaml.dump(build_info)

  for p in paths:
    logging.info("Writing build information to %s", p)
    if p.startswith("gs://"):
      if not gcs_client:
        gcs_client = storage.Client(project=project)
      bucket_name, path = util.split_gcs_uri(p)
      bucket = gcs_client.get_bucket(bucket_name)
      blob = bucket.blob(path)
      blob.upload_from_string(contents)

    else:
      with open(p, mode='w') as hf:
        hf.write(contents)


def build(args):
  """Build the code."""
  if not args.src_dir:
    logging.info("--src_dir not set")
    args.src_dir = os.path.abspath(
      os.path.join(os.path.dirname(__file__), ".."))
  logging.info("Use --src_dir=%s", args.src_dir)

  go_dir = os.getenv("GOPATH")
  if not go_dir:
    raise ValueError("Environment variable GOPATH must be set.")

  go_src_dir = os.path.join(go_dir, "src", "github.com", REPO_ORG, REPO_NAME)

  if not os.path.exists(go_src_dir):
    logging.info("%s does not exist.", go_src_dir)

    # Create a symbolic link in the go path.
    parent_dir = os.path.dirname(go_src_dir)
    if not os.path.exists(parent_dir):
      os.makedirs(parent_dir)
    logging.info("Creating symbolic link %s pointing to %s", go_src_dir,
                 args.src_dir)
    os.symlink(args.src_dir, go_src_dir)

  # Check that the directory in the go src path correctly points to
  # the same directory as args.src_dir
  if os.path.islink(go_src_dir):
    target = os.path.realpath(go_src_dir)

    if target != args.src_dir:
      message = "{0} is a symbolic link to {1}; but --src_dir={2}".format(
        go_src_dir, target, args.src_dir)
      logging.error(message)
      raise ValueError(message)
  elif go_src_dir != args.src_dir:
    message = "{0} doesn't equal --src_dir={1}".format(go_src_dir, args.src_dir)
    logging.error(message)
    raise ValueError(message)

  vendor_dir = os.path.join(args.src_dir, "vendor")
  if not os.path.exists(vendor_dir):
    logging.info("Installing go dependencies")
    util.install_go_deps(args.src_dir)
  else:
    logging.info("vendor directory exists; not installing go dependencies.")

  # TODO(jlewi): We can stop passing go_dir because we not rely on go_dir
  # being set in the environment.
  build_and_push(go_dir, args.src_dir, args)


def build_and_push(go_dir, src_dir, args):
  if args.dryrun:
    logging.info("dryrun...")
    # In dryrun mode we want to produce the build info file because this
    # is needed to test xcoms with Airflow.
    if args.build_info_path:
      paths = [args.build_info_path]
      build_info = {
        "image": "gcr.io/dryrun/dryrun:latest",
        "commit": "1234abcd",
        "helm_package": "gs://dryrun/dryrun.latest.",
      }
      write_build_info(build_info, paths, project=args.project)
    return
  build_and_push_artifacts(
    go_dir,
    src_dir,
    registry=args.registry,
    gcb_project=args.project,
    build_info_path=args.build_info_path,
    version_tag=args.version_tag)


def build_local(args):
  """Build the artifacts from the local copy of the code."""
  go_dir = os.getenv("GOPATH")
  if not go_dir:
    raise ValueError("GOPATH environment variable must be set.")

  src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))

  go_src_dir = os.path.join(go_dir, "src", "github.com", REPO_ORG, REPO_NAME)

  if not os.path.exists(go_src_dir):
    logging.info("Directory %s  doesn't exist.", go_src_dir)
    logging.info("Creating symbolic link %s pointing to %s", go_src_dir,
                 src_dir)
    os.symlink(src_dir, go_src_dir)

  build_and_push(go_dir, src_dir, args)


def clone_repo(args):
  args.clone_func(args)


def clone_pr(args):
  branches = ["pull/{0}/head:pr".format(args.pr)]
  util.clone_repo(args.src_dir, REPO_ORG, REPO_NAME, args.commit, branches)


def clone_postsubmit(args):
  util.clone_repo(args.src_dir, REPO_ORG, REPO_NAME, args.commit)


# TODO(jlewi): Delete this function once
# https://github.com/kubeflow/tf-operator/issues/189 is fixed.
def build_commit(args, branches):
  top_dir = args.src_dir or tempfile.mkdtemp(prefix="tmpTFJobSrc")
  logging.info("Top level directory for source: %s", top_dir)

  go_dir = os.path.join(top_dir, "go")
  os.environ["GOPATH"] = go_dir
  logging.info("Temporary go_dir: %s", go_dir)

  clone_dir = os.path.join(top_dir, REPO_DIR)
  src_dir = os.path.join(go_dir, "src", "github.com", REPO_ORG, REPO_NAME)

  util.clone_repo(clone_dir, REPO_ORG, REPO_NAME, args.commit, branches)

  # Create a symbolic link in the go path.
  os.makedirs(os.path.dirname(src_dir))
  logging.info("Creating symbolic link %s pointing to %s", src_dir, clone_dir)
  os.symlink(clone_dir, src_dir)
  util.install_go_deps(clone_dir)
  build_and_push(go_dir, src_dir, args)


# TODO(jlewi): Delete this function once
# https://github.com/kubeflow/tf-operator/issues/189 is fixed.
def build_postsubmit(args):
  """Build the artifacts from a postsubmit."""
  build_commit(args, None)


# TODO(jlewi): Delete this function once
# https://github.com/kubeflow/tf-operator/issues/189 is fixed.
def build_pr(args):
  """Build the artifacts from a postsubmit."""
  branches = ["pull/{0}/head:pr".format(args.pr)]
  build_commit(args, branches)


def clone_lastgreen(args):
  gcs_client = storage.Client()
  sha = get_latest_green_presubmit(gcs_client)

  util.clone_repo(args.src_dir, util.MASTER_REPO_OWNER, util.MASTER_REPO_NAME,
                  sha)


def build_new_release(args):  # pylint: disable=too-many-locals
  """Find the latest release and build the artifacts if they are newer then
  the current release.
  """
  if not args.src_dir:
    raise ValueError("src_dir must be provided when building last green.")

  gcs_client = storage.Client()
  sha = get_latest_green_presubmit(gcs_client)

  bucket_name, _ = util.split_gcs_uri(args.releases_path)
  bucket = gcs_client.get_bucket(bucket_name)

  logging.info("Latest passing postsubmit is %s", sha)

  last_release_sha = get_last_release(bucket)
  logging.info("Most recent release was for %s", last_release_sha)

  sha = build_and_push_image.GetGitHash(args.src_dir)

  if sha == last_release_sha:
    logging.info("Already cut release for %s", sha)
    return

  build(args)


def add_common_args(parser):
  """Add a set of common parser arguments."""

  parser.add_argument(
    "--registry",
    default="gcr.io/kubeflow-ci",
    type=str,
    help="The docker registry to use.")

  parser.add_argument(
    "--project",
    default=None,
    type=str,
    help=("If specified use Google Container Builder and this project to "
          "build artifacts."))

  parser.add_argument(
    "--releases_path",
    default=None,
    required=False,
    type=str,
    help="The GCS location where artifacts should be pushed.")

  parser.add_argument(
    "--build_info_path",
    default="",
    type=str,
    help="(Optional). The GCS location to write build info to.")

  parser.add_argument(
    "--version_tag",
    default=None,
    type=str,
    help=("A string used as the image tag. If not supplied defaults to a "
          "value based on the git commit."))

  parser.add_argument(
    "--dryrun", dest="dryrun", action="store_true", help="Do a dry run.")
  parser.add_argument(
    "--no-dryrun",
    dest="dryrun",
    action="store_false",
    help="Don't do a dry run.")
  parser.set_defaults(dryrun=False)


def build_parser():
  # create the top-level parser
  parser = argparse.ArgumentParser(description="Build the release artifacts.")
  subparsers = parser.add_subparsers()

  #############################################################################
  # clone
  #
  # Create the parser for the "local" mode.
  # This mode builds the artifacts from the local copy of the code.

  parser_clone = subparsers.add_parser(
    "clone", help="Clone and checkout the repository.")

  parser_clone.add_argument(
    "--src_dir",
    required=True,
    type=str,
    help="Directory to checkout the source to.")

  clone_subparsers = parser_clone.add_subparsers()

  last_green = clone_subparsers.add_parser(
    "lastgreen", help="Clone the last green postsubmit.")

  last_green.add_argument(
    "--commit",
    default=None,
    type=str,
    help="Optional a particular commit to checkout.")

  last_green.set_defaults(clone_func=clone_lastgreen)

  pr = clone_subparsers.add_parser("pr", help="Clone the pull request.")

  pr.add_argument(
    "--pr", default=None, required=True, help="The pull request to check out..")

  pr.add_argument(
    "--commit",
    default=None,
    type=str,
    help="Optional a particular commit to checkout.")

  pr.set_defaults(clone_func=clone_pr)

  postsubmit = clone_subparsers.add_parser(
    "postsubmit", help="Clone a postsubmit.")

  postsubmit.add_argument(
    "--commit",
    default=None,
    type=str,
    help="Optional a particular commit to checkout.")

  postsubmit.set_defaults(clone_func=clone_postsubmit)

  parser_clone.set_defaults(func=clone_repo)

  ############################################################################
  # Build command
  build_subparser = subparsers.add_parser("build", help="Build the artifacts.")

  build_subparser.add_argument(
    "--src_dir",
    default=None,
    type=str,
    help=("Directory containing the source. If not set determined "
          "automatically."))

  add_common_args(build_subparser)
  build_subparser.set_defaults(func=build)

  #############################################################################
  # local
  #
  # Create the parser for the "local" mode.
  # This mode builds the artifacts from the local copy of the code.

  parser_local = subparsers.add_parser(
    "local", help="Build the artifacts from the local copy of the code.")

  add_common_args(parser_local)
  parser_local.set_defaults(func=build_local)

  # Build a particular postsubmit hash.
  parser_postsubmit = subparsers.add_parser(
    "postsubmit", help="Build the artifacts from a postsubmit.")
  parser_postsubmit.set_defaults(func=build_postsubmit)

  add_common_args(parser_postsubmit)

  parser_postsubmit.add_argument(
    "--commit",
    default=None,
    type=str,
    help="Optional a particular commit to checkout and build.")

  parser_postsubmit.add_argument(
    "--src_dir",
    default=None,
    type=str,
    help="(Optional) Directory to checkout the source to.")

  ############################################################################
  # Build new release
  build_new = subparsers.add_parser(
    "build_new_release",
    help=("Build a new release. Only builds it if its newer than current "
          "release."))

  build_new.add_argument(
    "--src_dir",
    default=None,
    type=str,
    help=("Directory containing the source. "))

  add_common_args(build_new)
  build_new.set_defaults(func=build_new_release)

  ############################################################################
  # Pull Request
  parser_pr = subparsers.add_parser(
    "pr", help=("Build the artifacts from the specified pull request. "))

  add_common_args(parser_pr)

  parser_pr.add_argument(
    "--pr", required=True, type=str, help="The PR to build.")

  parser_pr.add_argument(
    "--commit",
    default=None,
    type=str,
    help="Optional a particular commit to checkout and build.")

  parser_pr.add_argument(
    "--src_dir",
    default=None,
    type=str,
    help="(Optional) Directory to checkout the source to.")

  parser_pr.set_defaults(func=build_pr)
  return parser


def main():  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals
  logging.basicConfig(
    level=logging.INFO,
    format=('%(levelname)s|%(asctime)s'
            '|%(pathname)s|%(lineno)d| %(message)s'),
    datefmt='%Y-%m-%dT%H:%M:%S',
  )

  util.maybe_activate_service_account()

  parser = build_parser()

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  # TODO: this line fails in Python 3 because library API change.
  args.func(args)


if __name__ == "__main__":
  main()
