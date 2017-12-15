# Copyright 2017 The TensorFlow Agents Authors.
#
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

"""Script to train a batch reinforcement learning algorithm.

Command line:

  python3 -m agents.scripts.train --logdir=/path/to/logdir --config=pendulum
"""

from __future__ import absolute_import, division, print_function

import datetime
import functools
import json
import os
import pprint
import re
import time

import gym
import tensorflow as tf
from agents import tools
from agents.scripts import configs, utility
from tensorflow.python.ops import variables

from . import loop as lp
from . import in_graph_batch_env, simulate


def define_batch_env(constructor, num_agents, env_processes):
  """Create environments and apply all desired wrappers.

  Args:
    constructor: Constructor of an OpenAI gym environment.
    num_agents: Number of environments to combine in the batch.
    env_processes: Whether to step environment in external processes.

  Returns:
    In-graph environments object.
  """
  with tf.variable_scope('environments'):
    if env_processes:
      envs = [
          tools.wrappers.ExternalProcess(constructor)
          for _ in range(num_agents)]
    else:
      envs = [constructor() for _ in range(num_agents)]
    batch_env = tools.BatchEnv(envs, blocking=not env_processes)
    batch_env = in_graph_batch_env.InGraphBatchEnv(batch_env)
  return batch_env


def define_saver(exclude=None):
  """Create a saver for the variables we want to checkpoint.

  Args:
    exclude: List of regexes to match variable names to exclude.

  Returns:
    Saver object.
  """
  variables = []
  exclude = exclude or []
  exclude = [re.compile(regex) for regex in exclude]
  for variable in tf.global_variables():
    if any(regex.match(variable.name) for regex in exclude):
      continue
    variables.append(variable)
  saver = tf.train.Saver(variables, keep_checkpoint_every_n_hours=5)
  return saver


def define_simulation_graph(batch_env, algo_cls, config, step):
  """Define the algortihm and environment interaction.

  Args:
    batch_env: In-graph environments object.
    algo_cls: Constructor of a batch algorithm.
    config: Configuration object for the algorithm.

  Returns:
    Object providing graph elements via attributes.
  """
  # pylint: disable=unused-variable
  is_training = tf.placeholder(tf.bool, name='is_training')
  should_log = tf.placeholder(tf.bool, name='should_log')
  do_report = tf.placeholder(tf.bool, name='do_report')
  force_reset = tf.placeholder(tf.bool, name='force_reset')

  algo = algo_cls(batch_env, step, is_training, should_log, config)

  done, score, summary = simulate.simulate(
      batch_env, algo, should_log, force_reset)

  message = 'Graph contains {} trainable variables.'
  tf.logging.info(message.format(tools.count_weights()))
  # pylint: enable=unused-variable
  return tools.AttrDict(locals())


def _create_environment(config):
  """Constructor for an instance of the environment.

  Args:
    config: Object providing configurations via attributes.

  Returns:
    Wrapped OpenAI Gym environment.
  """
  if isinstance(config.env, str):
    env = gym.make(config.env)
  else:
    env = config.env()
  if config.max_length:
    env = tools.wrappers.LimitDuration(env, config.max_length)
  env = tools.wrappers.RangeNormalize(env)
  env = tools.wrappers.ClipAction(env)
  env = tools.wrappers.ConvertTo32Bit(env)
  return env


def _define_loop(graph, logdir, train_steps, eval_steps):
  """Create and configure a training loop with training and evaluation phases.

  Args:
    graph: Object providing graph elements via attributes.
    logdir: Log directory for storing checkpoints and summaries.
    train_steps: Number of training steps per epoch.
    eval_steps: Number of evaluation steps per epoch.

  Returns:
    Loop object.
  """
  loop = lp.Loop(
      logdir, graph.step, graph.should_log, graph.do_report,
      graph.force_reset)
  loop.add_phase(
      'train', graph.done, graph.score, graph.summary, train_steps,
      report_every=train_steps,
      log_every=train_steps // 2,
      checkpoint_every=None,
      feed={graph.is_training: True})
  loop.add_phase(
      'eval', graph.done, graph.score, graph.summary, eval_steps,
      report_every=eval_steps,
      log_every=eval_steps // 2,
      checkpoint_every=10 * eval_steps,
      feed={graph.is_training: False})
  return loop


class RunConfig(object):
  def __init__(self):
    log_base = '== RunConfig.'
    self.cluster = json.loads(os.environ.get(
        "TF_CONFIG", "{}")).get("cluster", {})
    self.cluster = self._convert_keys_to_string(self.cluster)
    tf.logging.debug(log_base + 'cluster: %s' % self.cluster)
    self.task = json.loads(os.environ.get("TF_CONFIG", "{}")).get("task", {})
    tf.logging.debug(log_base + 'task: %s' % self.task)
    self.is_chief = (self.task['type'] == 'master')
    tf.logging.debug(log_base + 'is_chief: %s' % self.is_chief)
    self.job_name = str(self.task['type'])
    tf.logging.debug(log_base + 'job_name: %s' % self.job_name)
    self.task_index = (0 if 'index' not in self.task
                       else self.task['index'])
    tf.logging.debug(log_base + 'task_index: %s' % self.task_index)
    self.master = 'grpc://localhost:2222'
    if not (self.task['type'] is 'master'):
      self.master = 'grpc://%s' % self.cluster['master'][0]
    tf.logging.debug(log_base + 'master: %s' % self.master)
    self.cluster_spec = tf.train.ClusterSpec(self.cluster)
    self.cluster_def = self.cluster_spec.as_custer_def()
    self.server_def = tensorflow_server_pb2.ServerDef(
        cluster=self.cluster_def, job_name=self.job_name,
        task_index=self.task_index, protocol="grpc")
    tf.logging.info('server def: %s' % self.server_def)

  def _convert_keys_to_string(self, dictionary):
    """Recursively converts dictionary keys to strings."""
    if not isinstance(dictionary, dict):
      return dictionary
    return dict((str(k), self._convert_keys_to_string(v))
                for k, v in dictionary.items())


def _log_run_config(run_config):
  stem = '== RunConfig: '
  attrs = ['task_type', 'task_id', 'cluster_spec', 'master', 'num_ps_replicas',
           'num_worker_replicas', 'is_chief']

  for attr in attrs:
    if hasattr(run_config, attr):
      tf.logging.info(stem + '%s, %s' % (attr, getattr(run_config, attr)))


def _hack_default_to_local():
  def _collection_move_matching(pattern, source, target):
    source_collection_actual = tf.get_collection_ref(source)
    target_collection_actual = tf.get_collection_ref(target)
    for var in tf.get_collection(source):
      if re.match(pattern, var.name):
        target_collection_actual.append(var)
        source_collection_actual.remove(var)

  def _dump_collection(collection):
    tf.logging.debug('Dumping variables in collection %s' % collection)
    for var in tf.get_collection_ref(collection):
      tf.logging.debug('%s, %s' % (collection, str(var)))

  _dump_collection(tf.GraphKeys.GLOBAL_VARIABLES)
  _collection_move_matching(r'.', tf.GraphKeys.GLOBAL_VARIABLES,
                            tf.GraphKeys.LOCAL_VARIABLES)
  _collection_move_matching(r'network/policy', tf.GraphKeys.LOCAL_VARIABLES,
                            tf.GraphKeys.GLOBAL_VARIABLES)
  _collection_move_matching(r'network/value', tf.GraphKeys.LOCAL_VARIABLES,
                            tf.GraphKeys.GLOBAL_VARIABLES)
  _collection_move_matching(r'global_step', tf.GraphKeys.LOCAL_VARIABLES,
                            tf.GraphKeys.GLOBAL_VARIABLES)
  _dump_collection(tf.GraphKeys.GLOBAL_VARIABLES)
  tf.logging.debug('===')
  _dump_collection(tf.GraphKeys.LOCAL_VARIABLES)


def train(agents_config, env_processes, log_dir=None):
  """Training and evaluation entry point yielding scores.

  Resolves some configuration attributes, creates environments, graph, and
  training loop. By default, assigns all operations to the CPU.

  Args:
    config: Object providing configurations via attributes.
    env_processes: Whether to step environments in separate processes.

  Yields:
    Evaluation scores.
  """

  FLAGS = tf.app.flags.FLAGS

  if log_dir is None and hasattr(FLAGS, 'log_dir'):
    log_dir = FLAGS.log_dir

  run_config = tf.contrib.learn.RunConfig()

  _log_run_config(run_config)

  server = tf.train.Server(
      run_config.cluster_spec, job_name=run_config.task_type, task_index=run_config.task_id)

  tf.reset_default_graph()

  if agents_config.update_every % agents_config.num_agents:
    tf.logging.warn('Number of agents should divide episodes per update.')

  worker_device = "/job:%s/replica:0/task:%d" % (run_config.task_type,
                                                 run_config.task_id)

  with tf.device(worker_device):

    with tf.device('/job:ps/replica:0/task:0/device:CPU:0'):
      global_step = tf.Variable(0, False, dtype=tf.int32, name='global_step')

    batch_env = define_batch_env(
        lambda: _create_environment(agents_config),
        agents_config.num_agents, env_processes)

    graph = define_simulation_graph(
        batch_env, agents_config.algorithm, agents_config, global_step)

    loop = _define_loop(
        graph, log_dir,
        agents_config.update_every * agents_config.max_length,
        agents_config.eval_episodes * agents_config.max_length)

    total_steps = int(
        agents_config.steps / agents_config.update_every *
        (agents_config.update_every + agents_config.eval_episodes))

    # Exclude episode related variables since the Python state of environments is
    # not checkpointed and thus new episodes start after resuming.
    saver = utility.define_saver(exclude=(r'.*_temporary/.*',))

    sess_config = tf.ConfigProto(allow_soft_placement=True)
    if FLAGS.log_device_placement:
      sess_config.log_device_placement = True

    sess_config.gpu_options.allow_growth = True

    init_op = tf.global_variables_initializer()
    local_init_op = tf.local_variables_initializer()

    _hack_default_to_local()

    hooks = [tf.train.StopAtStepHook(last_step=total_steps)]

    if FLAGS.sync_replicas:
      opt = graph.algo._optimizer
      ready_for_local_init_op = opt.ready_for_local_init_op
      local_init_op = opt.local_step_init_op
      if run_config.is_chief:
        local_init_op = opt.chief_init_op

      # Initial token and chief queue runners required by the sync_replicas mode
      chief_queue_runner = opt.get_chief_queue_runner()
      sync_init_op = opt.get_init_tokens_op()

      sync_replicas_hook = opt.make_session_run_hook(run_config.is_chief)
      hooks.append(sync_replicas_hook)

    scaffold = tf.train.Scaffold(
        saver=saver,
        init_op=init_op,
        local_init_op=local_init_op,
        # ready_for_local_init_op=ready_for_local_init_op
    )

    with tf.train.MonitoredTrainingSession(
            master=server.target,
            is_chief=run_config.is_chief,
            checkpoint_dir=log_dir,
            scaffold=scaffold,
            hooks=hooks,
            save_checkpoint_secs=FLAGS.save_checkpoint_secs,
            save_summaries_steps=None,
            save_summaries_secs=None,
            config=sess_config,
            stop_grace_period_secs=120,
            log_step_count_steps=3000) as sess:

      global_step = sess.run(loop._step)
      steps_made = 1

      while not sess.should_stop():

        phase, epoch, steps_in = loop._find_current_phase(global_step)
        phase_step = epoch * phase.steps + steps_in

        if steps_in % phase.steps < steps_made:
          message = '\n' + ('-' * 50) + '\n'
          message += 'Phase {} (phase step {}, global step {}).'
          tf.logging.info(message.format(phase.name, phase_step, global_step))

        phase.feed[loop._reset] = (steps_in < steps_made)

        phase.feed[loop._log] = (
            phase.writer and
            loop._is_every_steps(phase_step, phase.batch, phase.log_every))
        phase.feed[loop._report] = (
            loop._is_every_steps(phase_step, phase.batch, phase.report_every))

        summary, mean_score, global_step, steps_made = sess.run(
            phase.op, phase.feed)

        if loop._is_every_steps(phase_step, phase.batch, phase.report_every):
          yield mean_score

        # TODO: Potentially integrate summary writing with
        # MonitoredTrainingSession.
        if summary and phase.writer and run_config.is_chief:
          # We want smaller phases to catch up at the beginnig of each epoch so
          # that their graphs are aligned.
          longest_phase = max(phase.steps for phase in loop._phases)
          summary_step = epoch * longest_phase + steps_in
          phase.writer.add_summary(summary, summary_step)

    batch_env.close()
