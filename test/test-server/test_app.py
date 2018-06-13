"""A simple flask server to be used in tests.

The purpose of this flask app is to allow us to control the behavior 
of various processes so that we can test the controller semantics.
"""

"""
Simple app that parses predictions from a trained model and displays them.
"""

import os
import sys
import random

import requests
from flask import Flask, json, render_template, request, g, jsonify

APP = Flask(__name__)

exit_code = None

@APP.route("/")
def index():
  """Default route.

  Placeholder, does nothing.
  """
  return "hello world"

@APP.route("/exit", methods=['GET'])
def exit():
  # Exit with the provided exit code
  global exit_code
  exit_code = int(request.args.get('exitCode', 0))
  shutdown_server()
  return "Shutting down with exitCode {0}".format(exit_code)

def shutdown_server():
  func = request.environ.get('werkzeug.server.shutdown')
  if func is None:
    raise RuntimeError('Not running with the Werkzeug Server')
  func()
  
if __name__ == '__main__':
  APP.run(debug=False, host='0.0.0.0', port=8080)
  sys.exit(exit_code)