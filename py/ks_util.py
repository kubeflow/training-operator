"""Utility for ksonnet operations."""
import filelock
import logging
import os
import re
import subprocess
from kubeflow.testing import util


def setup_ks_app(app_dir, env, namespace, component, params):
  """Setup the ksonnet app"""

  lock_file = os.path.join(app_dir, "app.lock")
  logging.info("Acquiring lock on file: %s", lock_file)
  lock = filelock.FileLock(lock_file, timeout=60)
  with lock:
    # Create a new environment for this run
    try:
      util.run(["ks", "env", "add", env, "--namespace=" + namespace],
                cwd=app_dir)
    except subprocess.CalledProcessError as e:
      if not re.search(".*environment.*already exists.*", e.output):
        raise

    for pair in params.split(","):
      k, v = pair.split("=", 1)
      util.run(
        ["ks", "param", "set", "--env=" + env, component, k, v],
        cwd=app_dir)
