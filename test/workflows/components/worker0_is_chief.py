#!/usr/local/bin/python
#

"""A simple python script that is used to test the exit policy
when worker 0 is the chief. In this case worker 0 exits
but no other workers do.
"""
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
    # TODO(jlewi): This is a hack to ensure the job runs long enough
    # for all pods/services to be created. A better approach would be to
    # actually detect the creation of the pods/services. There's a couple
    # approaches we could take:
    # 1. We could use the same logic as test_runner.py to detect the K8s
    #    events and wait for the expected events.
    # 2. We could run a server and test_runner could issue an RPC when
    #    the test should continue.
    # #2 Is more flexible as that approach would allow us to write more
    # powerful E2E tests for failure handling and restart behavior.
    time.sleep(30)
    logging.info("Worker 0 is chief; exiting.")
  else:
    logging.info("Non-chief process runs forever")
    while True:
      logging.info("Run forever")
      time.sleep(30)

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  main()
