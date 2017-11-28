"""Train a simple TF program to verify we can execute ops.

The program does a simple matrix multiplication.

Only the master assigns ops to devices/workers.

The master will assign ops to every task in the cluster. This way we can verify
that distributed training is working by executing ops on all devices.
"""
import argparse
import json
import logging
import os

import tensorflow as tf


def parse_args():
  """Parse the command line arguments."""
  parser = argparse.ArgumentParser()

  parser.add_argument(
      "--sleep_secs",
      default=0,
      type=int,
      help=("Amount of time to sleep at the end"))

  # TODO(jlewi): We ignore unknown arguments because the backend is currently
  # setting some flags to empty values like metadata path.
  args, _ = parser.parse_known_args()
  return args


def run(server, cluster_spec):  # pylint: disable=too-many-statements, too-many-locals
  """Build the graph and run the example.

  Args:
    server: The TensorFlow server to use.

  Raises:
    RuntimeError: If the expected log entries aren't found.
  """

  # construct the graph and create a saver object
  with tf.Graph().as_default():  # pylint: disable=not-context-manager
    # The initial value should be such that type is correctly inferred as
    # float.
    width = 10
    height = 10
    results = []

    # The master assigns ops to every TFProcess in the cluster.
    for job_name in cluster_spec.keys():
      for i in range(len(cluster_spec[job_name])):
        d = "/job:{0}/task:{1}".format(job_name, i)
        with tf.device(d):
          a = tf.constant(range(width * height), shape=[height, width])
          b = tf.constant(range(width * height), shape=[height, width])
          c = tf.multiply(a, b)
          results.append(c)

    init_op = tf.global_variables_initializer()

    if server:
      target = server.target
    else:
      # Create a direct session.
      target = ""

    logging.info("Server target: %s", target)
    with tf.Session(
            target, config=tf.ConfigProto(log_device_placement=True)) as sess:
      sess.run(init_op)
      for r in results:
        result = sess.run(r)
        logging.info("Result: %s", result)


def main():
  """Run training.

  Raises:
    ValueError: If the arguments are invalid.
  """
  logging.info("Tensorflow version: %s", tf.__version__)
  logging.info("Tensorflow git version: %s", tf.__git_version__)

  tf_config_json = os.environ.get("TF_CONFIG", "{}")
  tf_config = json.loads(tf_config_json)
  logging.info("tf_config: %s", tf_config)

  task = tf_config.get("task", {})
  logging.info("task: %s", task)

  cluster_spec = tf_config.get("cluster", {})
  logging.info("cluster_spec: %s", cluster_spec)

  server = None
  device_func = None
  if cluster_spec:
    cluster_spec_object = tf.train.ClusterSpec(cluster_spec)
    server_def = tf.train.ServerDef(
        cluster=cluster_spec_object.as_cluster_def(),
        protocol="grpc",
        job_name=task["type"],
        task_index=task["index"])

    logging.info("server_def: %s", server_def)

    logging.info("Building server.")
    # Create and start a server for the local task.
    server = tf.train.Server(server_def)
    logging.info("Finished building server.")

    # Assigns ops to the local worker by default.
    device_func = tf.train.replica_device_setter(
        worker_device="/job:worker/task:%d" % server_def.task_index,
        cluster=server_def.cluster)
  else:
    # This should return a null op device setter since we are using
    # all the defaults.
    logging.error("Using default device function.")
    device_func = tf.train.replica_device_setter()

  job_type = task.get("type", "").lower()
  if job_type == "ps":
    logging.info("Running PS code.")
    server.join()
  elif job_type == "worker":
    logging.info("Running Worker code.")
    # The worker just blocks because we let the master assign all ops.
    server.join()
  elif job_type == "master" or not job_type:
    logging.info("Running master.")
    with tf.device(device_func):
      run(server=server, cluster_spec=cluster_spec)
  else:
    raise ValueError("invalid job_type %s" % (job_type,))


if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  main()
