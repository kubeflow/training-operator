# How to debug an E2E test for Kubeflow Training Operator

[E2E Testing](./e2e_testing.md) gives an overview of writing e2e tests. This guidance concentrates more on the e2e failure debugging.


## Prerequsite

1. Install python 3.7

2. Clone `kubeflow/testing` repo under `$GOPATH/src/kubeflow/`

3. Install [ksonnet](https://ksonnet.io/)

```
wget https://github.com/ksonnet/ksonnet/releases/download/v0.13.1/ks_0.13.1_linux_amd64.tar.gz
tar -xvzf ks_0.13.1_linux_amd64.tar.gz
sudo cp ks_0.13.1_linux_amd64/ks /usr/local/bin/ks-13
```
> We would like to deprecate `ksonnet` but may takes some time. Feel free to pick up [the issue](https://github.com/kubeflow/training-operator/issues/1468) if you are interested in it. 
> If your platform is darwin or windows, feel free to download binaries in [ksonnet v0.13.1](https://github.com/ksonnet/ksonnet/releases/tag/v0.13.1)

4. Deploy HEAD training operator version in your environment

```
IMG=kubeflow/training-operator:e2e-debug-prid make docker-build

# Optional - load image into kind cluster if you are using kind
kind load docker-image kubeflow/training-operator:e2e-debug-1462

kubectl set image deployment.v1.apps/training-operator training-operator=kubeflow/training-operator:e2e-debug-1462
```

## Run E2E Tests locally

1. Set environments
```
export KUBEFLOW_PATH=$GOPATH/src/github.com/kubeflow
export KUBEFLOW_TRAINING_REPO=$KUBEFLOW_PATH/training-operator
export KUBEFLOW_TESTING_REPO=$KUBEFLOW_PATH/testing
export PYTHONPATH=$KUBEFLOW_TRAINING_REPO:$KUBEFLOW_TRAINING_REPO/py:$KUBEFLOW_TESTING_REPO/py:$KUBEFLOW_TRAINING_REPO/sdk/python
```


2. Install python dependencies
```
pip3 install -r $KUBEFLOW_TESTING_REPO/py/kubeflow/testing/requirements.txt
```

> Note: if you have meet problem install requirement, you may need to `sudo apt-get install libffi-dev`. Feel free to share error logs if you don't know how to handle it.


3. Run Tests
```
# enter the ksonnet app to run tests
cd $KUBEFLOW_TRAINING_REPO/test/workflows

# run individual test that failed in the presubmit job.
python3 -m kubeflow.tf_operator.pod_names_validation_tests --app_dir=$KUBEFLOW_TRAINING_REPO/test/workflows --params=name=pod-names-validation-tests-v1,namespace=kubeflow --tfjob_version=v1 --num_trials=1 --artifacts_path=/tmp/output/artifacts
python3 -m kubeflow.tf_operator.cleanpod_policy_tests --app_dir=$KUBEFLOW_TRAINING_REPO/test/workflows --params=name=cleanpod-policy-tests-v1,namespace=kubeflow --tfjob_version=v1 --num_trials=1 --artifacts_path=/tmp/output/artifacts
python3 -m kubeflow.tf_operator.simple_tfjob_tests  --app_dir=$KUBEFLOW_TRAINING_REPO/test/workflows --params=name=simple-tfjob-tests-v1,namespace=kubeflow --tfjob_version=v1 --num_trials=2 --artifacts_path=/tmp/output/artifact
```


## Check results

You can either check logs or check results in `/tmp/output/artifact`. 

```
$ ls -al /tmp/output/artifact
junit_test_simple_tfjob_cpu.xml

$ cat /tmp/output/artifact/junit_test_simple_tfjob_cpu.xml
<testsuite failures="0" tests="1" time="659.5505294799805"><testcase classname="SimpleTfJobTests" name="simple-tfjob-tests-v1" time="659.5505294799805" /></testsuite>
```

## Common issues

1. ksonnet is not installed 

```
ERROR|2021-11-16T03:06:06|/home/jiaxin.shan/go/src/github.com/kubeflow/training-operator/py/kubeflow/tf_operator/test_runner.py|57| There was a problem running the job; Exception [Errno 2] No such file or directory: 'ks-13': 'ks-13'
Traceback (most recent call last):
  File "/home/jiaxin.shan/go/src/github.com/kubeflow/training-operator/py/kubeflow/tf_operator/test_runner.py", line 38, in run_test
    test_func()
  File "/home/jiaxin.shan/go/src/github.com/kubeflow/training-operator/py/kubeflow/tf_operator/pod_names_validation_tests.py", line 53, in test_pod_names
    self.params)
  File "/home/jiaxin.shan/go/src/github.com/kubeflow/training-operator/py/kubeflow/tf_operator/util.py", line 579, in setup_ks_app
    cwd=app_dir)
  File "/home/jiaxin.shan/go/src/github.com/kubeflow/testing/py/kubeflow/testing/util.py", line 59, in run
    command, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
  File "/usr/local/lib/python3.7/subprocess.py", line 775, in __init__
    restore_signals, start_new_session)
  File "/usr/local/lib/python3.7/subprocess.py", line 1522, in _execute_child
    raise child_exception_type(errno_num, err_msg, err_filename)
FileNotFoundError: [Errno 2] No such file or directory: 'ks-13': 'ks-13'
```

Please check `Prerequsite` section to install ksonnet.


2. TypeError: load() missing 1 required positional argument: 'Loader'

```
ERROR|2021-11-16T03:04:12|/home/jiaxin.shan/go/src/github.com/kubeflow/training-operator/py/kubeflow/tf_operator/test_runner.py|57| There was a problem running the job; Exception load() missing 1 required positional argument: 'Loader'
Traceback (most recent call last):
  File "/home/jiaxin.shan/go/src/github.com/kubeflow/training-operator/py/kubeflow/tf_operator/test_runner.py", line 38, in run_test
    test_func()
  File "/home/jiaxin.shan/go/src/github.com/kubeflow/training-operator/py/kubeflow/tf_operator/pod_names_validation_tests.py", line 51, in test_pod_names
    ks_cmd = ks_util.get_ksonnet_cmd(self.app_dir)
  File "/home/jiaxin.shan/go/src/github.com/kubeflow/testing/py/kubeflow/testing/ks_util.py", line 47, in get_ksonnet_cmd
    results = yaml.load(app_yaml)
TypeError: load() missing 1 required positional argument: 'Loader'
```

This is the pyyaml compatibility issue. Please check if you are using pyyaml==6.0.0. If so, downgrade to `5.4.1` instead.

```
pip3 uninstall pyyaml
pip3 install pyyaml==5.4.1 --user
```
