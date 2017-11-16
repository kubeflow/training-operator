from __future__ import absolute_import, division, print_function

import argparse
import datetime
import logging
import os

import agents
import pybullet_envs  # To make AntBulletEnv-v0 available.
import tensorflow as tf


def pybullet_ant():
  # General
  algorithm = agents.ppo.PPOAlgorithm
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
  #optimizer = 'AdamOptimizer'
  optimizer = tf.train.AdamOptimizer
  learning_rate = 1e-4
  # Losses
  discount = 0.995
  kl_target = 1e-2
  kl_cutoff_factor = 2
  kl_cutoff_coef = 1000
  kl_init_penalty = 1
  return locals()


def main(args):
  agents.scripts.utility.set_up_logging()
  log_dir = args.log_dir and os.path.expanduser(args.log_dir)
  if log_dir:
    log_dir = os.path.join(
        log_dir, '{}-{}'.format(args.timestamp, args.config))
  if args.mode == 'train':
    try:
      # Try to resume training.
      config = agents.scripts.utility.load_config(args.log_dir)
    except IOError:
      # Start new training run.
      config = agents.tools.AttrDict(globals()[args.config]())
      config = agents.scripts.utility.save_config(config, log_dir)
    for score in agents.scripts.train.train(config, env_processes=True):
      logging.info('Score {}.'.format(score))
  if args.mode == 'render':
    agents.scripts.visualize.visualize(
        logdir=args.log_dir, outdir=args.log_dir, num_agents=1, num_episodes=5,
        checkpoint=None, env_processes=True)


if __name__ == '__main__':
  timestamp = datetime.datetime.now().strftime('%Y%m%dT%H%M%S')
  parser = argparse.ArgumentParser()
  parser.add_argument('--mode', choices=['train', 'render'], default='train')
  parser.add_argument('--log_dir', default=None, required=True)
  parser.add_argument('--config', default=None, required=True)
  parser.add_argument('--timestamp', default=timestamp)
  args = parser.parse_args()
  main(args)
