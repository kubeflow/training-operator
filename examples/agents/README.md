# Tensorflow/agents on k8s

Demonstration of training reinforcement learning agents from [tensorflow/agents](https://github.com/tensorflow/agents) on kubernetes using the tf/k8s CRD.

One approach to run the example is to use the [example runner](https://github.com/tensorflow/k8s/tree/master/py/k8s) to (1) build and deploy the model container and (2) construct, submit, and monitor the job, for example as follows:

```bash
k8s --mode submit --build_path ${SCRIPT_DIR} \
    --extra_args '--config pybullet_ant'
```

Alternatively you may wish to build and deploy images and submit TfJob operations some other way, perhaps building the CRD config manually, in which case the following job config example may be useful:

```yaml
apiVersion: "tensorflow.org/v1alpha1"
kind: "TfJob"
metadata:
  name: "<your job name>"
  namespace: default
spec:
  replicaSpecs:
    - replicas: 1
      tfReplicaType: MASTER
      template:
        spec:
          containers:
            - image: <repository with worker image>
              name: tensorflow
              args: ['--log_dir', 'gs://<bucket path to log dir>', '--config', 'pybullet_ant']
          restartPolicy: OnFailure
  tensorBoard:
    logDir: gs://<bucket path to log dir>
```

In this example hyperparameters are provided in [trainer/task.py](https://github.com/tensorflow/k8s/tree/master/examples/agents/trainer/task.py):

```python
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
```
