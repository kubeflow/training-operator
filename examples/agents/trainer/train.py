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
import os

import gym
import tensorflow as tf
from agents import tools
from agents.scripts import configs, utility


def define_simulation_graph(batch_env, algo_cls, config):
  """Define the algortihm and environment interaction.

  Args:
    batch_env: In-graph environments object.
    algo_cls: Constructor of a batch algorithm.
    config: Configuration object for the algorithm.

  Returns:
    Object providing graph elements via attributes.
  """
  # pylint: disable=unused-variable
  run_config = tf.contrib.learn.RunConfig()

  # Obtain the worker device function
  # device_func = tf.train.replica_device_setter(
  #     worker_device="/job:worker/task:%d" % run_config.task_id,
  #     ps_device="/job:ps/cpu:0",
  #     cluster=run_config.cluster_spec)

  device_func = tf.train.replica_device_setter(
      worker_device="/job:%s/task:%d" % (run_config.task_type,
                                         run_config.task_id),
      ps_device="/job:ps/cpu:0",
      cluster=run_config.cluster_spec)

  with tf.device(device_func):
    # step = tf.get_variable('global_step', [],
    #                        initializer=tf.constant_initializer(0),
    #                        trainable=False)
    step = tf.Variable(0, False, dtype=tf.int32, name='global_step')

  # step = tf.Variable(0, False, dtype=tf.int32, name='global_step')

  is_training = tf.placeholder(tf.bool, name='is_training')
  should_log = tf.placeholder(tf.bool, name='should_log')
  do_report = tf.placeholder(tf.bool, name='do_report')
  force_reset = tf.placeholder(tf.bool, name='force_reset')
  algo = algo_cls(batch_env, step, is_training, should_log, config)
  done, score, summary = tools.simulate(
      batch_env, algo, should_log, force_reset)
  message = 'Graph contains {} trainable variables.'
  tf.logging.info(message.format(tools.count_weights()))
  # pylint: enable=unused-variable
  return tools.AttrDict(locals())


def initialize_variables(sess, saver, logdir, checkpoint=None, resume=None):
  """Initialize or restore variables from a checkpoint if available.

  Args:
    sess: Session to initialize variables in.
    saver: Saver to restore variables.
    logdir: Directory to search for checkpoints.
    checkpoint: Specify what checkpoint name to use; defaults to most recent.
    resume: Whether to expect recovering a checkpoint or starting a new run.

  Raises:
    ValueError: If resume expected but no log directory specified.
    RuntimeError: If no resume expected but a checkpoint was found.
  """
  sess.run(tf.group(
      tf.local_variables_initializer(),
      tf.global_variables_initializer()))
  if resume and not (logdir or checkpoint):
    raise ValueError('Need to specify logdir to resume a checkpoint.')
  if logdir:
    state = tf.train.get_checkpoint_state(logdir)
    if checkpoint:
      checkpoint = os.path.join(logdir, checkpoint)
    if not checkpoint and state and state.model_checkpoint_path:
      checkpoint = state.model_checkpoint_path
    if checkpoint and resume is False:
      message = 'Found unexpected checkpoint when starting a new run.'
      raise RuntimeError(message)
    if checkpoint:
      saver.restore(sess, checkpoint)


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
  loop = tools.Loop(
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


def _run_loop(loop, sess, saver, max_step=None):
  """Run the loop schedule for a specified number of steps.

  Call the operation of the current phase until the global step reaches the
  specified maximum step. Phases are repeated over and over in the order they
  were added.

  Args:
    sess: Session to use to run the phase operation.
    saver: Saver used for checkpointing.
    max_step: Run the operations until the step reaches this limit.

  Yields:
    Reported mean scores.
  """
  global_step = sess.run(loop._step)
  steps_made = 1
  while True:
    if max_step and global_step >= max_step:
      break
    phase, epoch, steps_in = loop._find_current_phase(global_step)
    phase_step = epoch * phase.steps + steps_in
    if steps_in % phase.steps < steps_made:
      message = '\n' + ('-' * 50) + '\n'
      message += 'Phase {} (phase step {}, global step {}).'
      tf.logging.info(message.format(phase.name, phase_step, global_step))
    # Populate book keeping tensors.
    phase.feed[loop._reset] = (steps_in < steps_made)
    phase.feed[loop._log] = (
        phase.writer and
        loop._is_every_steps(phase_step, phase.batch, phase.log_every))
    phase.feed[loop._report] = (
        loop._is_every_steps(phase_step, phase.batch, phase.report_every))
    summary, mean_score, global_step, steps_made = sess.run(
        phase.op, phase.feed)
    if loop._is_every_steps(phase_step, phase.batch, phase.checkpoint_every):
      loop._store_checkpoint(sess, saver, global_step)
    if loop._is_every_steps(phase_step, phase.batch, phase.report_every):
      yield mean_score
    if summary and phase.writer:
      # We want smaller phases to catch up at the beginnig of each epoch so
      # that their graphs are aligned.
      longest_phase = max(phase.steps for phase in loop._phases)
      summary_step = epoch * longest_phase + steps_in
      phase.writer.add_summary(summary, summary_step)


def train(agents_config, run_config, log_dir, env_processes):
  """Training and evaluation entry point yielding scores.

  Resolves some configuration attributes, creates environments, graph, and
  training loop. By default, assigns all operations to the CPU.

  Args:
    config: Object providing configurations via attributes.
    env_processes: Whether to step environments in separate processes.

  Yields:
    Evaluation scores.
  """

  server_def = tf.train.ServerDef(
      cluster=run_config.cluster_spec.as_cluster_def(),
      protocol="grpc",
      job_name=run_config.task_type,
      task_index=run_config.task_id)

  server = tf.train.Server(server_def)

  device_func = tf.train.replica_device_setter(
      worker_device="/job:worker/task:%d" % run_config.task_id,
      cluster=run_config.cluster_spec)

  tf.reset_default_graph()
  if agents_config.update_every % agents_config.num_agents:
    tf.logging.warn('Number of agents should divide episodes per update.')

  # with tf.device('/cpu:0'):
  with tf.device("/job:%s/task:%d/cpu:0" % (run_config.task_type,
                                            run_config.task_id)):
    # with tf.device(device_func):
    # Don't want to share all tf.Variable's since rollouts are run to obtain
    # experience independently?

    batch_env = utility.define_batch_env(
        lambda: _create_environment(agents_config),
        agents_config.num_agents, env_processes)

    graph = define_simulation_graph(
        batch_env, agents_config.algorithm, agents_config)

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
    # saver = None
    # device_filters = ["/job:ps",
    #                   "/job:worker/task:%d" % run_config.task_id]
    sess_config = tf.ConfigProto(allow_soft_placement=True,
                                 # device_filters=device_filters,
                                 log_device_placement=True
                                 )
    sess_config.gpu_options.allow_growth = True

    # - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    # Get ops and variables needed for use of SyncReplicasOptimizer

    init_op = tf.global_variables_initializer()

    # Get a reference to the optimizer from the graph
    opt = graph.algo._optimizer
    global_step = graph.step

    # Get a reference to the propper local_init_op
    local_init_op = opt.local_step_init_op
    if run_config.is_chief:
      local_init_op = opt.chief_init_op

    # Get init ops to be run on chief queue runner
    chief_queue_runner = opt.get_chief_queue_runner()
    sync_init_op = opt.get_init_tokens_op()

    ready_for_local_init_op = opt.ready_for_local_init_op

    # - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    # Initialize the training supervisor with necessary params for run and init
    # and sync of replicas.
    # Handles checkpointing and resuming from previous checkpoint.
    # sv = tf.train.Supervisor(
    #     is_chief=run_config.is_chief,
    #     logdir=log_dir,
    #     init_op=init_op,
    #     local_init_op=local_init_op,
    #     ready_for_local_init_op=ready_for_local_init_op,
    #     recovery_wait_secs=1,
    #     global_step=global_step)
    #
    # # If master, create session, otherwise wait for session to be created.
    # sess = sv.prepare_or_wait_for_session(server.target, config=sess_config)
    #
    # if run_config.is_chief:
    #   # Chief worker will start the chief queue runner and call the init op.
    #   sess.run(sync_init_op)
    #   sv.start_queue_runners(sess, [chief_queue_runner])

    # Traceback (most recent call last):
    #   File "/usr/lib/python2.7/runpy.py", line 174, in _run_module_as_main
    #     "__main__", fname, loader, pkg_name)
    #   File "/usr/lib/python2.7/runpy.py", line 72, in _run_code
    #     exec code in run_globals
    #   File "/app/trainer/task.py", line 158, in <module>
    #     main(args)
    #   File "/app/trainer/task.py", line 144, in main
    #     for score in train.train(agents_config, run_config, args.log_dir, env_processes=True):
    #   File "trainer/train.py", line 305, in train
    #     sess = sv.prepare_or_wait_for_session(server.target, config=sess_config)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/supervisor.py", line 716, in prepare_or_wait_for_session
    #     max_wait_secs=max_wait_secs)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 400, in wait_for_session
    #     sess)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 481, in _try_run_local_init_op
    #     is_ready_for_local_init, msg = self._model_ready_for_local_init(sess)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 466, in _model_ready_for_local_init
    #     "Model not ready for local init")
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 508, in _ready
    #     ready_value = sess.run(op)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 895, in run
    #     run_metadata_ptr)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 1109, in _run
    #     self._graph, fetches, feed_dict_tensor, feed_handles=feed_handles)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 426, in __init__
    #     self._assert_fetchable(graph, fetch.op)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 439, in _assert_fetchable
    #     'Operation %r has been marked as not fetchable.' % op.name)
    # ValueError: Operation u'end_episode/cond/cond/training/scan_1/while/report_uninitialized_variables/boolean_mask/Gather' has been marked as not fetchable.

    # - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    # scaffold = tf.train.Scaffold(
    #     saver=saver,
    #     init_op=init_op,
    #     ready_op=sync_init_op,  # ??
    #     ready_for_local_init_op=ready_for_local_init_op,
    #     local_init_op=local_init_op,
    #     summary_op=None,
    #     init_feed_dict=None,
    #     init_fn=None
    # )
    #
    # # TODO: User MonitoredTrainingSession since Supervisor is headed for depr.
    # hooks = [tf.train.StopAtStepHook(last_step=total_steps)]
    # sess = tf.train.MonitoredTrainingSession(
    #     master=server.target,
    #     is_chief=run_config.is_chief,
    #     checkpoint_dir=log_dir,
    #     scaffold=scaffold,
    #     hooks=hooks,
    #     chief_only_hooks=None,
    #     save_checkpoint_secs=600,
    #     # save_summaries_steps=USE_DEFAULT,
    #     # save_summaries_secs=USE_DEFAULT,
    #     config=sess_config,
    #     stop_grace_period_secs=120,
    #     log_step_count_steps=100
    # )

    #     Traceback (most recent call last):
    #   File "/usr/lib/python2.7/runpy.py", line 174, in _run_module_as_main
    #     "__main__", fname, loader, pkg_name)
    #   File "/usr/lib/python2.7/runpy.py", line 72, in _run_code
    #     exec code in run_globals
    #   File "/app/trainer/task.py", line 158, in <module>
    #     main(args)
    #   File "/app/trainer/task.py", line 144, in main
    #     for score in train.train(agents_config, run_config, args.log_dir, env_processes=True):
    #   File "trainer/train.py", line 376, in train
    #     log_step_count_steps=100
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/monitored_session.py", line 336, in MonitoredTrainingSession
    #     stop_grace_period_secs=stop_grace_period_secs)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/monitored_session.py", line 668, in __init__
    #     stop_grace_period_secs=stop_grace_period_secs)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/monitored_session.py", line 490, in __init__
    #     self._sess = _RecoverableSession(self._coordinated_creator)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/monitored_session.py", line 842, in __init__
    #     _WrappedSession.__init__(self, self._create_session())
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/monitored_session.py", line 847, in _create_session
    #     return self._sess_creator.create_session()
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/monitored_session.py", line 551, in create_session
    #     self.tf_sess = self._session_creator.create_session()
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/monitored_session.py", line 460, in create_session
    #     max_wait_secs=30 * 60  # Wait up to 30 mins for the session to be ready.
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 400, in wait_for_session
    #     sess)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 481, in _try_run_local_init_op
    #     is_ready_for_local_init, msg = self._model_ready_for_local_init(sess)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 466, in _model_ready_for_local_init
    #     "Model not ready for local init")
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/training/session_manager.py", line 508, in _ready
    #     ready_value = sess.run(op)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 895, in run
    #     run_metadata_ptr)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 1109, in _run
    #     self._graph, fetches, feed_dict_tensor, feed_handles=feed_handles)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 426, in __init__
    #     self._assert_fetchable(graph, fetch.op)
    #   File "/usr/local/lib/python2.7/dist-packages/tensorflow/python/client/session.py", line 439, in _assert_fetchable
    #     'Operation %r has been marked as not fetchable.' % op.name)
    # ValueError: Operation u'end_episode/cond/cond/training/scan_1/while/report_uninitialized_variables/boolean_mask/Gather' has been marked as not fetchable.

    # - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    # initialize_variables(sess, saver, log_dir)
    # for score in loop.run(sess, saver, total_steps):
    #   yield score

    # When running with SyncReplicasOptimizer without using opt.* ops via
    # Supervisor, hangs, presumably because it's stuck waiting for a sync
    # operation?
    with tf.Session(server.target, config=sess_config) as sess:
      utility.initialize_variables(sess, saver, log_dir)
      for score in loop.run(sess, saver, total_steps):
        yield score

    batch_env.close()


def main(_):
  """Create or load configuration and launch the trainer."""
  utility.set_up_logging()
  if not FLAGS.config:
    raise KeyError('You must specify a configuration.')
  logdir = FLAGS.logdir and os.path.expanduser(os.path.join(
      FLAGS.logdir, '{}-{}'.format(FLAGS.timestamp, FLAGS.config)))
  try:
    config = utility.load_config(logdir)
  except IOError:
    config = tools.AttrDict(getattr(configs, FLAGS.config)())
    config = utility.save_config(config, logdir)
  for score in train_runner(config, FLAGS.env_processes):
    tf.logging.info('Score {}.'.format(score))


if __name__ == '__main__':
  FLAGS = tf.app.flags.FLAGS
  tf.app.flags.DEFINE_string(
      'logdir', None,
      'Base directory to store logs.')
  tf.app.flags.DEFINE_string(
      'timestamp', datetime.datetime.now().strftime('%Y%m%dT%H%M%S'),
      'Sub directory to store logs.')
  tf.app.flags.DEFINE_string(
      'config', None,
      'Configuration to execute.')
  tf.app.flags.DEFINE_boolean(
      'env_processes', True,
      'Step environments in separate processes to circumvent the GIL.')
  tf.app.run()
