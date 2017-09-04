# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Site level customizations.

This script is imported by python when any python script is run.
https://docs.python.org/2/library/site.html.
This script configures python logging to write stderr and stdout to a
rotating log file.
"""

import json
import logging
import logging.handlers
import os
import sys
import traceback

# TODO(jlewi): User's would need to have pythonjsonlogger and wrapt installed in their container.
# Can we find a way to avoid creating extra dependencies for the user?
#
# https://github.com/madzak/python-json-logger
from pythonjsonlogger import jsonlogger
from wrapt import register_post_import_hook

def exception_handler(exctype, value, tb):
  """Handler for uncaught exceptions.

  We want to log uncaught exceptions using the logger so that we properly
  format it for export to Cloud Logging.

  Args:
    exctype: Type of the exception.
    value: The exception.
    tb: The traceback.
  """
  logging.exception(''.join(traceback.format_exception(exctype, value, tb)))


class TfJobJsonFormatter(jsonlogger.JsonFormatter):
  """Formats log lines in JSON.
  """
  def __init__(self):
    # TODO(jlewi): We need to get additional labels from an environment variable.
    super(TfJobJsonFormatter, self).__init__(
      '%(name)s|%(levelname)s|%(message)s|%(created)f'
      '|%(lineno)d|%(pathname)s', '%Y-%m-%dT%H:%M:%S')

  def _replace_with_original(self, log_record, key):
    """Replaces log_record[key] with log_record['original_' + key]."""
    original_key = 'original_%s' % key
    if original_key in log_record:
      log_record[key] = log_record[original_key]
      del log_record[original_key]

  def process_log_record(self, log_record):
    """Modifies fields in the log_record to match fluentd's expectations."""
    self._replace_with_original(log_record, 'created')
    self._replace_with_original(log_record, 'pathname')
    self._replace_with_original(log_record, 'lineno')
    self._replace_with_original(log_record, 'thread')

    log_record['severity'] = log_record['levelname']
    log_record['timestampSeconds'] = int(log_record['created'])
    log_record['timestampNanos'] = int(
        (log_record['created'] % 1) * 1000 * 1000 * 1000)

    return log_record


def get_log_level():
  """Reads the log level from environment variable CLOUD_ML_LOG_LEVEL."""
  log_level = logging.INFO
  log_level_env_variable = 'LOG_LEVEL'
  if log_level_env_variable in os.environ:
    log_level_name = os.environ[log_level_env_variable]
    if log_level_name == 'DEBUG':
      log_level = logging.DEBUG
    elif log_level_name == 'WARNING':
      log_level = logging.WARNING
    elif log_level_name == 'ERROR':
      log_level = logging.ERROR
    elif log_level_name == 'CRITICAL':
      log_level = logging.CRITICAL
  return log_level


def configure_logger():
  """Configures logging to write JSON log entries to a rotating file.
  """

  # Configure the root logger, ensuring that we always have at least one
  # handler.
  root_logger = logging.getLogger()
  root_logger_previous_handlers = list(root_logger.handlers)
  for h in root_logger_previous_handlers:
    root_logger.removeHandler(h)
  root_logger.setLevel(get_log_level())

  handler = logging.StreamHandler()
  handler.setFormatter(TfJobJsonFormatter())
  root_logger.addHandler(handler)

  # Insert exception handler for uncaught exceptions.
  sys.excepthook = exception_handler

  # Ensure that we reconfigure the 'tensorflow' logger after TensorFlow
  # configures it.
  register_post_import_hook(reconfigure_tf_logger, 'tensorflow')


def reconfigure_tf_logger(unused_module):
  """Removes the 'tensorflow' logger handlers.

  Logging via the tf_logging module will be propagated up the logger hierarchy
  to the root_logger configured above.
  """
  tf_logger = logging.getLogger('tensorflow')
  while tf_logger.handlers:
    tf_logger.removeHandler(tf_logger.handlers[0])

# Turn on logging
configure_logger()


