#!/usr/local/bin/python
#

"""A simple python script that is used to test the exit policy
when worker 0 is the chief. In this case worker 0 exits
but no other workers do.
"""
import argparse
import json
import logging
import os
import time

def main():
  """Run training.
  Raises:
    ValueError: If the arguments are invalid.
  """
  tf_config_json = os.environ.get("TF_CONFIG", "{}")
  tf_config = json.loads(tf_config_json)
  logging.info("tf_config: %s", tf_config)

  task = tf_config.get("task", {})
  logging.info("task: %s", task)

  job_name = task["type"]
  task_index = task["index"]

  if job_name == "worker" and task_index == 0:
    logging.info("Worker 0 is chief; exiting.")
  else:
    logging.info("Non-chief process runs forever")
    while True:
      logging.info("Run forever")
      time.sleep(30)

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  main()
