from __future__ import absolute_import, division, print_function

import argparse
import datetime
import logging
import os

import tensorflow as tf

import agents
import algorithm
import pybullet_envs  # To make AntBulletEnv-v0 available.


def pybullet_ant():
  """Hyperparameter configuration written by save_config and loaded by train."""

  # General
  algorithm = algorithm.PPOAlgorithm
  num_agents = 10
  eval_episodes = 25
  use_gpu = False
  # Environment
  env = 'AntBulletEnv-v0'
  max_length = 1000
  steps = 1e7  # 10M
  # Network
  network = agents.scripts.networks.feed_forward_gaussian
  weight_summaries = dict(
      all=r'.*',
      policy=r'.*/policy/.*',
      value=r'.*/value/.*')
  policy_layers = 200, 100
  value_layers = 200, 100
  init_mean_factor = 0.1
  init_logstd = -1
  # Optimization
  update_every = 30
  update_epochs = 25
  # optimizer = 'AdamOptimizer'
  optimizer = tf.train.AdamOptimizer
  learning_rate = 1e-4
  # Losses
  discount = 0.995
  kl_target = 1e-2
  kl_cutoff_factor = 2
  kl_cutoff_coef = 1000
  kl_init_penalty = 1
  return locals()


def _get_training_configuration(config_var_name, log_dir):
  """Load hyperparameter config."""
  try:
    # Try to resume training.
    config = agents.scripts.utility.load_config(log_dir)
  except IOError:
    # Load hparams from object in globals() by name.
    config = agents.tools.AttrDict(globals()[config_var_name]())
    # Write the hyperparameters for this run to a config YAML for posteriority
    config = agents.scripts.utility.save_config(config, log_dir)

  return config


def _get_distributed_configuration():
  """Get a tensorflow cluster spec and task objects based on TF_CONFIG var."""
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

  return server, device_func, task


def _train(args):
  """Run in training mode."""
  config = _get_training_configuration(args.config, args.log_dir)
  server, worker_device_func, task = _get_distributed_configuration()
  job_type = task.get("type", "").lower()

  if job_type == "ps":
    logging.info("Running PS.")
    server.join()
  else:
    logging.info("Running worker.")
    # Runs agents.scripts.train.train on each of the workers. Within those,
    # trainer.algorithm.PPOAlgorithm is called, each running a replica of the
    # training graph on the local node and (TODO) sharing policy parameters
    # by placing these on parameter servers. The shared policy parameters are
    # then asynchronously updated by each worker with the gradients they
    # compute based on the rollouts performed in the environments local to that
    # worker.
    with tf.device(device_func):
      for score in agents.scripts.train.train(config, env_processes=True):
        logging.info('Score {}.'.format(score))


def _render(args):
  """Run in render mode"""
  raise NotImplementedError()
  # # Read model checkpoint from args.log_dir and render num_agents for
  # # num_episodes.
  # agents.scripts.visualize.visualize(
  #     logdir=args.log_dir, outdir=args.log_dir, num_agents=1, num_episodes=5,
  #     checkpoint=None, env_processes=True)


def main():
  """Run training.

  Raises:
    ValueError: If the arguments are invalid.
  """
  logging.info("Tensorflow version: %s", tf.__version__)
  logging.info("Tensorflow git version: %s", tf.__git_version__)

  agents.scripts.utility.set_up_logging()
  log_dir = args.log_dir and os.path.expanduser(args.log_dir)
  if log_dir:
    args.log_dir = os.path.join(
      log_dir, '{}-{}'.format(args.timestamp, args.config))

  if args.mode == 'render':
    _render(args)

  if args.mode == 'train':
    _train(args)


if __name__ == '__main__':
  timestamp = datetime.datetime.now().strftime('%Y%m%dT%H%M%S')
  parser = argparse.ArgumentParser()
  parser.add_argument('--mode', choices=['train', 'render'], default='train')
  parser.add_argument('--log_dir', default=None, required=True)
  parser.add_argument('--config', default=None, required=True)
  parser.add_argument('--timestamp', default=timestamp)
  args = parser.parse_args()
  main(args)
