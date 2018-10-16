"""Utility for ksonnet operations."""
import filelock
import logging
import os
import re
import subprocess
import uuid
from kubeflow.testing import util


def setup_ks_app(args):
  """Setup the ksonnet app"""
  salt = uuid.uuid4().hex[0:4]

  lock_file = os.path.join(args.app_dir, "app.lock")
  logging.info("Acquiring lock on file: %s", lock_file)
  lock = filelock.FileLock(lock_file, timeout=60)
  with lock:
    # Create a new environment for this run
    if "environment" in args and args.environment:
      env = args.environment
    else:
      env = "test-env-{0}".format(salt)

    name = None
    namespace = None
    for pair in args.params.split(","):
      k, v = pair.split("=", 1)
      if k == "name":
        name = v

      if k == "namespace":
        namespace = v

    if not name:
      raise ValueError("name must be provided as a parameter.")

    if not namespace:
      raise ValueError("namespace must be provided as a parameter.")

    try:
      util.run(["ks", "env", "add", env, "--namespace=" + namespace],
                cwd=args.app_dir)
    except subprocess.CalledProcessError as e:
      if not re.search(".*environment.*already exists.*", e.output):
        raise

    for pair in args.params.split(","):
      k, v = pair.split("=", 1)
      util.run(
        ["ks", "param", "set", "--env=" + env, args.component, k, v],
        cwd=args.app_dir)

    return namespace, name, env

  return "", "", ""

