"""Run checks on python source files.

This binary invokes checks (e.g. lint and unittests) on our Python source files.
"""
import argparse
import fnmatch
import logging
import os
import subprocess
import time

from google.cloud import storage  # pylint: disable=no-name-in-module

from py import util
from py import test_util


def run_lint(args):
  start_time = time.time()
  # Print out the pylint version because different versions can produce
  # different results.
  util.run(["pylint", "--version"])

  # kubeflow_testing is imported as a submodule so we should exclude it
  # TODO(jlewi): Perhaps we should get a list of submodules and exclude
  # them automatically?
  dir_excludes = [
    "dashboard/frontend/node_modules",
    "kubeflow_testing",
    "test/test-app",
    "vendor",
  ]
  full_dir_excludes = [
    os.path.join(os.path.abspath(args.src_dir), f) for f in dir_excludes
  ]
  includes = ["*.py"]
  failed_files = []
  rc_file = os.path.join(args.src_dir, ".pylintrc")
  for root, dirs, files in os.walk(os.path.abspath(args.src_dir), topdown=True):
    # excludes can be done with fnmatch.filter and complementary set,
    # but it's more annoying to read.
    exclude = False
    for e in full_dir_excludes:
      if root.startswith(e):
        exclude = True
        break
    if exclude:
      continue

    dirs[:] = [d for d in dirs]
    for pat in includes:
      for f in fnmatch.filter(files, pat):
        full_path = os.path.join(root, f)
        try:
          util.run(
            ["pylint", "--rcfile=" + rc_file, full_path], cwd=args.src_dir)
        except subprocess.CalledProcessError:
          failed_files.append(full_path[len(args.src_dir):])

  if failed_files:
    failed_files.sort()
    logging.error("%s files had lint errors:\n%s", len(failed_files),
                  "\n".join(failed_files))
  else:
    logging.info("No lint issues.")

  if not args.junit_path:
    logging.info("No --junit_path.")
    return

  test_case = test_util.TestCase()
  test_case.class_name = "pylint"
  test_case.name = "pylint"
  test_case.time = time.time() - start_time
  if failed_files:
    test_case.failure = "Files with lint issues: {0}".format(
      ", ".join(failed_files))

  gcs_client = None
  if args.junit_path.startswith("gs://"):
    gcs_client = storage.Client(project=args.project)

  test_util.create_junit_xml_file([test_case], args.junit_path, gcs_client)


def run_tests(args):
  # Print out the pylint version because different versions can produce
  # different results.
  util.run(["pylint", "--version"])

  # kubeflow_testing is imported as a submodule so we should exclude it
  # TODO(jlewi): Perhaps we should get a list of submodules and exclude
  # them automatically?
  dir_excludes = ["kubeflow_testing", "vendor"]
  includes = ["*_test.py"]
  test_cases = []

  env = os.environ.copy()
  # TODO(jlewi): Once we switch to using Argo I think we can stop setting
  # the PYTHONPATH here and just inheriting it from the environment.
  # When we use ARGO each step will run in its own pod and we can set the
  # PYTHONPATH environment variable as needed for that pod.
  env["PYTHONPATH"] = (
    args.src_dir + ":" + os.path.join(args.src_dir, "kubeflow_testing", "py"))

  num_failed = 0
  for root, dirs, files in os.walk(args.src_dir, topdown=True):
    # excludes can be done with fnmatch.filter and complementary set,
    # but it's more annoying to read.
    dirs[:] = [d for d in dirs if d not in dir_excludes]
    for pat in includes:
      for f in fnmatch.filter(files, pat):
        full_path = os.path.join(root, f)

        test_case = test_util.TestCase()
        test_case.class_name = "pytest"
        test_case.name = full_path[len(args.src_dir):]
        start_time = time.time()
        test_cases.append(test_case)
        try:
          util.run(["python", full_path], cwd=args.src_dir, env=env)
        except subprocess.CalledProcessError:
          test_case.failure = "{0} failed.".format(test_case.name)
          num_failed += 1
        finally:
          test_case.time = time.time() - start_time

  if num_failed:
    logging.error("%s tests failed.", num_failed)
  else:
    logging.info("No lint issues.")

  if not args.junit_path:
    logging.info("No --junit_path.")
    return

  gcs_client = None
  if args.junit_path.startswith("gs://"):
    gcs_client = storage.Client(project=args.project)

  test_util.create_junit_xml_file(test_cases, args.junit_path, gcs_client)


def add_common_args(parser):
  """Add a set of common parser arguments."""

  parser.add_argument(
    "--src_dir",
    default=os.getcwd(),
    type=str,
    help=("The root directory of the source tree. Defaults to current "
          "directory."))

  parser.add_argument(
    "--project",
    default=None,
    type=str,
    help=("(Optional). The project to use with the GCS client."))

  parser.add_argument(
    "--junit_path",
    default=None,
    type=str,
    help=("(Optional). The GCS location to write the junit file with the "
          "results."))


def main():  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals
  logging.basicConfig(
    level=logging.INFO,
    format=('%(levelname)s|%(asctime)s'
            '|%(pathname)s|%(lineno)d| %(message)s'),
    datefmt='%Y-%m-%dT%H:%M:%S',
  )
  # create the top-level parser
  parser = argparse.ArgumentParser(description="Run python code checks.")
  subparsers = parser.add_subparsers()

  #############################################################################
  # lint
  #
  # Create the parser for running lint.

  parser_lint = subparsers.add_parser("lint", help="Run lint.")

  add_common_args(parser_lint)
  parser_lint.set_defaults(func=run_lint)

  #############################################################################
  # tests
  #
  # Create the parser for running the tests.

  parser_test = subparsers.add_parser("test", help="Run tests.")

  add_common_args(parser_test)
  parser_test.set_defaults(func=run_tests)

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)
  logging.info("Finished")


if __name__ == "__main__":
  main()
