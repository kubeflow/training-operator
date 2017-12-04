# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Example illustrating training of PPO agent using tensorflow/k8s CRD."""

from __future__ import absolute_import, division, print_function

import argparse
import datetime
import json
import logging
import os

import agents
import tensorflow as tf

import pybullet_envs  # To make AntBulletEnv-v0 available.
from trainer.algorithm import PPOAlgorithm

from . import train
from .networks import feed_forward_gaussian


def smoke():
  """Hyperparameter configuration written by save_config and loaded by train."""

  # General
  algorithm = PPOAlgorithm
  num_agents = 1
  eval_episodes = 2
  use_gpu = False
  # Environment
  env = 'AntBulletEnv-v0'
  max_length = 1000
  steps = 10  # 10M
  # Network
  network = feed_forward_gaussian
  weight_summaries = dict(
      all=r'.*',
      policy=r'.*/policy/.*',
      value=r'.*/value/.*')
  policy_layers = 2, 1
  value_layers = 2, 1
  init_mean_factor = 0.1
  init_logstd = -1
  # Optimization
  update_every = 1
  update_epochs = 1
  optimizer = tf.train.AdamOptimizer
  learning_rate = 1e-4
  # Losses
  discount = 0.995
  kl_target = 1e-2
  kl_cutoff_factor = 2
  kl_cutoff_coef = 1000
  kl_init_penalty = 1
  return locals()


def pybullet_ant():
  """Hyperparameter configuration written by save_config and loaded by train."""

  # General
  algorithm = PPOAlgorithm
  num_agents = 10
  eval_episodes = 25
  use_gpu = False
  # Environment
  env = 'AntBulletEnv-v0'
  max_length = 1000
  steps = 1e7  # 10M
  # Network
  network = feed_forward_gaussian
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


def _get_agents_configuration(config_var_name, log_dir):
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


def main(args):
  """Run training.

  Raises:
    ValueError: If the arguments are invalid.
  """
  logging.info("Tensorflow version: %s", tf.__version__)
  logging.info("Tensorflow git version: %s", tf.__git_version__)

  agents.scripts.utility.set_up_logging()
  agents_config = _get_agents_configuration(args.config, args.log_dir)
  run_config = tf.contrib.learn.RunConfig()
  log_dir = args.log_dir and os.path.expanduser(args.log_dir)

  if log_dir:
    # HACK: This is really not what we want to do. Definitely want to only be
    # having master write logs.
    args.log_dir = os.path.join(
        log_dir, '{}-{}-tid{}-{}'.format(args.timestamp, run_config.task_type,
                                         run_config.task_id, args.config))
    ###

  if args.mode == 'train' or args.mode == 'smoke':
    # train.train_bk(agents_config, env_processes=True)
    for score in train.train(agents_config, run_config, args.log_dir, env_processes=True):
      tf.logging.info('Score {}.'.format(score))


if __name__ == '__main__':
  logging.getLogger().setLevel(logging.INFO)
  timestamp = datetime.datetime.now().strftime('%Y%m%dT%H%M%S')
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--mode', choices=['train', 'render'], default='train')
  parser.add_argument('--log_dir', default=None, required=True)
  parser.add_argument('--config', default=None, required=True)
  parser.add_argument('--timestamp', default=timestamp)
  args = parser.parse_args()
  main(args)
