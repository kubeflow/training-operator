"""A simple flask server to be used in tests.

The purpose of this flask app is to allow us to control the behavior
of various processes so that we can test the controller semantics.
"""
import argparse
import logging
import os
import sys

from flask import Flask, request

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
  # Exit with the provided exit code
  return os.environ.get("TF_CONFIG", "")

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
