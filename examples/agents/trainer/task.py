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
import pprint
import sys

import agents
import tensorflow as tf
#from . import networks
from agents.scripts import networks

import pybullet_envs
from trainer.algorithm import PPOAlgorithm

from . import train

flags = tf.app.flags

flags.DEFINE_string("mode", "train",
                    "Run mode, one of [train, visualize].")
flags.DEFINE_string("log_dir", None,
                    "The base directory in which to write logs and "
                    "checkpoints.")
flags.DEFINE_string("config", None,
                    "The name of the config object to be used to parameterize "
                    "the run.")
flags.DEFINE_string("run_base_tag",
                    datetime.datetime.now().strftime('%Y%m%dT%H%M%S'),
                    "Base tag to prepend to logs dir folder name. Defaults "
                    "to timestamp.")
flags.DEFINE_boolean("env_processes", True,
                     "Step environments in separate processes to circumvent "
                     "the GIL.")
flags.DEFINE_boolean("sync_replicas", False,
                     "Use the sync_replicas (synchronized replicas) mode, "
                     "wherein the parameter updates from workers are "
                     "aggregated before applied to avoid stale gradients.")
flags.DEFINE_integer("num_gpus", 0,
                     "Total number of gpus for each machine."
                     "If you don't use GPU, please set it to '0'")
flags.DEFINE_integer("save_checkpoint_secs", 600,
                     "Number of seconds between checkpoint save.")
flags.DEFINE_boolean("use_monitored_training_session", False,
                     "Whether to use tf.train.MonitoredTrainingSession to "
                     "manage the training session. If not, use "
                     "tf.train.Supervisor.")
flags.DEFINE_boolean("log_device_placement", False,
                     "Whether to output logs listing the devices on which "
                     "variables are placed.")
flags.DEFINE_boolean("debug", True,
                     "Use debugger to track down bad values during training")
flags.DEFINE_string("debug_ui_type", "curses",
                    "Command-line user interface type (curses | readline)")
FLAGS = flags.FLAGS


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
  steps = 1000  # 10M
  # Network
  network = networks.feed_forward_gaussian
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
  num_agents = 1
  eval_episodes = 25
  use_gpu = False
  # Environment
  env = 'AntBulletEnv-v0'
  max_length = 1000
  steps = 1e7  # 10M
  # Network
  network = networks.feed_forward_gaussian
  weight_summaries = dict(
      all=r'.*',
      policy=r'.*/policy/.*',
      value=r'.*/value/.*')
  # policy_layers = 20, 10
  policy_layers = 200, 100
  # value_layers = 20, 10
  value_layers = 200, 100
  init_mean_factor = 0.1
  init_logstd = -1
  # Optimization
  update_every = 30
  update_epochs = 25
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


def main(unused_argv):
  """Run training.

  Raises:
    ValueError: If the arguments are invalid.
  """
  tf.logging.set_verbosity(tf.logging.INFO)
  tf.logging.info("Tensorflow version: %s", tf.__version__)
  tf.logging.info("Tensorflow git version: %s", tf.__git_version__)

  if FLAGS.debug:
    tf.logging.set_verbosity(tf.logging.DEBUG)

  tf.logging.debug('FLAGS: \n %s' % pprint.pprint(FLAGS.__dict__))

  agents_config = _get_agents_configuration(FLAGS.config, FLAGS.log_dir)
  tf.logging.debug(pprint.pprint(agents_config))

  run_config = tf.contrib.learn.RunConfig()

  log_dir = FLAGS.log_dir and os.path.expanduser(FLAGS.log_dir)

  if log_dir:
    FLAGS.log_dir = os.path.join(
        log_dir, '{}-{}'.format(FLAGS.run_base_tag, FLAGS.config))

  if FLAGS.mode == 'train' or FLAGS.mode == 'smoke':
    for score in train.train(agents_config, env_processes=True):
      tf.logging.info('Mean score: %s' % score)


if __name__ == '__main__':
  tf.app.run()
