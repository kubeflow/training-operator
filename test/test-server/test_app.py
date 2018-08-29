"""A simple flask server to be used in tests.

The purpose of this flask app is to allow us to control the behavior
of various processes so that we can test the controller semantics.
"""
import argparse
import json
import logging
import os
import sys

from flask import Flask, request
from tensorflow.python.estimator import run_config as run_config_lib

APP = Flask(__name__)

exit_code = None

@APP.route("/")
def index():
  """Default route.

  Placeholder, does nothing.
  """
  return "hello world"

@APP.route("/tfconfig", methods=['GET'])
def tf_config():
  return os.environ.get("TF_CONFIG", "")

@APP.route("/runconfig", methods=['GET'])
def run_config():
  config = run_config_lib.RunConfig()
  if config:
    config_dict = {
      'master': config.master,
      'task_id': config.task_id,
      'num_ps_replicas': config.num_ps_replicas,
      'num_worker_replicas': config.num_worker_replicas,
      'cluster_spec': config.cluster_spec.as_dict(),
      'task_type': config.task_type,
      'is_chief': config.is_chief,
    }
    return json.dumps(config_dict)
  return ""

@APP.route("/exit", methods=['GET'])
def exitHandler():
  # Exit with the provided exit code
  global exit_code # pylint: disable=global-statement
  exit_code = int(request.args.get('exitCode', 0))
  shutdown_server()
  return "Shutting down with exitCode {0}".format(exit_code)

def shutdown_server():
  func = request.environ.get('werkzeug.server.shutdown')
  if func is None:
    raise RuntimeError('Not running with the Werkzeug Server')
  func()

if __name__ == '__main__':
  logging.getLogger().setLevel(logging.INFO)
  logging.basicConfig(
    level=logging.INFO,
    format=('%(levelname)s|%(asctime)s'
            '|%(pathname)s|%(lineno)d| %(message)s'),
    datefmt='%Y-%m-%dT%H:%M:%S',)

  parser = argparse.ArgumentParser(description="TFJob test server.")

  parser.add_argument(
    "--port",
    # By default use the same port as TFJob uses so that we can use the
    # TFJob services to address the test app.
    default=2222,
    type=int,
    help="The port to run on.")

  args = parser.parse_args()

  APP.run(debug=False, host='0.0.0.0', port=args.port)
  sys.exit(exit_code)
