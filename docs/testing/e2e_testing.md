# How to Write an E2E Test for Kubeflow Training Operator

The E2E tests for Kubeflow Training operator are implemented as Argo workflows. For more background and details
about Argo (not required for understanding the rest of this document), please take a look at
[this link](https://github.com/kubeflow/testing/blob/master/README.md).

Test results can be monitored at the [Prow dashboard](http://prow.kubeflow-testing.com/?repo=kubeflow%2Ftraining-operator).

At a high level, the E2E test suites are structured as Python test classes. Each test class contains
one or more tests. A test typically runs the following:

- Create a ksonnet component using a TFJob spec;
- Creates the specified TFJob;
- Verifies some expected results (e.g. number of pods started, job status);
- Deletes the TFJob.

## Adding a Test Method

An example can be found [here](https://github.com/kubeflow/training-operator/blob/master/py/kubeflow/tf_operator/simple_tfjob_tests.py).

A test class can have several test methods. Each method executes a series of user actions (e.g.
starting or deleting a TFJob), and performs verifications of expected results (e.g. TFJob exits with
correct status, pods are deleted, etc).

Test classes should follow this pattern:

```python
class MyTest(test_util.TestCase):
  def __init__(self, args):
    # Initialize environment

  def test_case_1(self):
    # Test code

  def test_case_2(self):
    # Test code

if __name__ == "__main__"
  test_runner.main(module=__name__)
```

The code here ideally should only contain API calls. Any common functionalities used by the test code should
be added to one of the helper modules:

- k8s_util - for K8s operations like querying/deleting a pod
- ks_util - for ksonnet operations
- tf_job_client - for TFJob-specific operations, such as waiting for the job to be in a certain phase

## Adding a TFJob Spec

This is needed if you want to use your own TFJob spec instead of an existing one. An example can be found
[here](https://github.com/kubeflow/training-operator/tree/master/test/workflows/components/simple_tfjob_v1.jsonnet).
All TFJob specs should be placed in the same directory.

These are similar to actual TFJob specs. Note that many of these are using the
[training-operator-test-server](https://github.com/kubeflow/training-operator/tree/master/test/test-server) as the test image.
This gives us more control over when each replica exits, and allows us to send specific requests like fetching the
runtime TensorFlow config.

## Adding a New Test Class

This is needed if you are creating a new test class. Creating a new test class is recommended if you are implementing
a new feature, and want to group all relevant E2E tests together.

New test classes should be added as Argo workflow steps to the
[workflows.libsonnet](https://github.com/kubeflow/training-operator/blob/master/test/workflows/components/workflows.libsonnet) file.

Under the templates section, add the following to the dag:

```
  {
    name: "my-test",
    template: "my-test",
    dependencies: ["setup-kubeflow"],
  },
```

This will configure Argo to run `my-test` after setting up the Kubeflow cluster.

Next, add the following lines toward the end of the file:

```
  $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
         "my-test"),
```

This assumes that there is a corresponding Python file named `my_test.py` (note the difference between dashes and
underscores).
