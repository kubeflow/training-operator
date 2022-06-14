# PyTorchJobClient

> PyTorchJobClient(config_file=None, context=None, client_configuration=None, persist_config=True)

User can loads authentication and cluster information from kube-config file and stores them in kubernetes.client.configuration. Parameters are as following:

parameter |  Description
------------ | -------------
config_file | Name of the kube-config file. Defaults to `~/.kube/config`. Note that for the case that the SDK is running in cluster and you want to operate PyTorchJob in another remote cluster, user must set `config_file` to load kube-config file explicitly, e.g. `PyTorchJobClient(config_file="~/.kube/config")`. |
context |Set the active context. If is set to None, current_context from config file will be used.|
client_configuration | The kubernetes.client.Configuration to set configs to.|
persist_config | If True, config file will be updated when changed (e.g GCP token refresh).|


The APIs for PyTorchJobClient are as following:

Class | Method |  Description
------------ | ------------- | -------------
PyTorchJobClient| [create](#create) | Create PyTorchJob|
PyTorchJobClient | [get](#get)    | Get the specified PyTorchJob or all PyTorchJob in the namespace |
PyTorchJobClient | [patch](#patch)  | Patch the specified PyTorchJob|
PyTorchJobClient | [delete](#delete) | Delete the specified PyTorchJob |
PyTorchJobClient | [wait_for_job](#wait_for_job) | Wait for the specified job to finish |
PyTorchJobClient | [wait_for_condition](#wait_for_condition) | Waits until any of the specified conditions occur |
PyTorchJobClient | [get_job_status](#get_job_status) | Get the PyTorchJob status|
PyTorchJobClient | [is_job_running](#is_job_running) | Check if the PyTorchJob running |
PyTorchJobClient | [is_job_succeeded](#is_job_succeeded) | Check if the PyTorchJob Succeeded |
PyTorchJobClient | [get_pod_names](#get_pod_names) | Get pod names of PyTorchJob |
PyTorchJobClient | [get_logs](#get_logs) | Get training logs of the PyTorchJob |

## create
> create(pytorchjob, namespace=None)

Create the provided pytorchjob in the specified namespace

### Example

```python
from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container
from kubernetes.client import V1ResourceRequirements

from kubeflow.training import constants
from kubeflow.training import utils
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1PyTorchJob
from kubeflow.training import KubeflowOrgV1PyTorchJobSpec
from kubeflow.training import PyTorchJobClient

  container = V1Container(
    name="pytorch",
    image="gcr.io/kubeflow-ci/pytorch-dist-mnist-test:v1.0",
    args=["--backend","gloo"],
  )

  master = V1ReplicaSpec(
    replicas=1,
    restart_policy="OnFailure",
    template=V1PodTemplateSpec(
      spec=V1PodSpec(
        containers=[container]
      )
    )
  )

  worker = V1ReplicaSpec(
    replicas=1,
    restart_policy="OnFailure",
    template=V1PodTemplateSpec(
      spec=V1PodSpec(
        containers=[container]
        )
    )
  )

  pytorchjob = KubeflowOrgV1PyTorchJob(
    api_version="kubeflow.org/v1",
    kind="PyTorchJob",
    metadata=V1ObjectMeta(name="mnist", namespace='default'),
    spec=KubeflowOrgV1PyTorchJobSpec(
      clean_pod_policy="None",
      pytorch_replica_specs={"Master": master,
                             "Worker": worker}
    )
  )

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.create(pytorchjob)

```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
pytorchjob  | [KubeflowOrgV1PyTorchJob](KubeflowOrgV1PyTorchJob.md) | pytorchjob defination| Required |
namespace | str | Namespace for pytorchjob deploying to. If the `namespace` is not defined, will align with pytorchjob definition, or use current or default namespace if namespace is not specified in pytorchjob definition.  | Optional |

### Return type
object

## get
> get(name=None, namespace=None, watch=False, timeout_seconds=600)

Get the created pytorchjob in the specified namespace

### Example

```python
from kubeflow.training import pytorchjobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.get('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | pytorchjob name. If the `name` is not specified, it will get all pytorchjobs in the namespace.| Optional. |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace.| Optional |
watch | bool | Watch the created pytorchjob if `True`, otherwise will return the created pytorchjob object. Stop watching if pytorchjob reaches the optional specified `timeout_seconds` or once the PyTorchJob status `Succeeded` or `Failed`. | Optional |
timeout_seconds | int | Timeout seconds for watching. Defaults to 600. | Optional |


### Return type
object


## patch
> patch(name, pytorchjob, namespace=None)

Patch the created pytorchjob in the specified namespace.

Note that if you want to set the field from existing value to `None`, `patch` API may not work, you need to use [replace](#replace) API to remove the field value.

### Example

```python

pytorchjob = KubeflowOrgV1PyTorchJob(
    api_version="kubeflow.org/v1",
    ... #update something in PyTorchJob spec
)

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.patch('mnist', isvc)

```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
pytorchjob  | [KubeflowOrgV1PyTorchJob](KubeflowOrgV1PyTorchJob.md) | pytorchjob defination| Required |
namespace | str | The pytorchjob's namespace for patching. If the `namespace` is not defined, will align with pytorchjob definition, or use current or default namespace if namespace is not specified in pytorchjob definition. | Optional|

### Return type
object


## delete
> delete(name, namespace=None)

Delete the created pytorchjob in the specified namespace

### Example

```python
from kubeflow.training import pytorchjobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.delete('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | pytorchjob name| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace. | Optional|

### Return type
object

## wait_for_job
> wait_for_job(name,
>              namespace=None,
>              watch=False,
>              timeout_seconds=600,
>              polling_interval=30,
>              status_callback=None):

Wait for the specified job to finish.

### Example

```python
from kubeflow.training import PyTorchJobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.wait_for_job('mnist', namespace='kubeflow')

# The API also supports watching the PyTorchJob status till it's Succeeded or Failed.
pytorchjob_client.wait_for_job('mnist', namespace='kubeflow', watch=True)
NAME                           STATE                TIME
pytorch-dist-mnist-gloo        Created              2020-01-02T09:21:22Z
pytorch-dist-mnist-gloo        Running              2020-01-02T09:21:36Z
pytorch-dist-mnist-gloo        Running              2020-01-02T09:21:36Z
pytorch-dist-mnist-gloo        Running              2020-01-02T09:21:36Z
pytorch-dist-mnist-gloo        Running              2020-01-02T09:21:36Z
pytorch-dist-mnist-gloo        Succeeded            2020-01-02T09:26:38Z
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The PyTorchJob name.| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace. | Optional|
watch | bool | Watch the PyTorchJob if `True`. Stop watching if PyTorchJob reaches the optional specified `timeout_seconds` or once the PyTorchJob status `Succeeded` or `Failed`. | Optional |
timeout_seconds | int | How long to wait for the job, default wait for 600 seconds. | Optional|
polling_interval | int | How often to poll for the status of the job.| Optional|
status_callback | str | Callable. If supplied this callable is invoked after we poll the job. Callable takes a single argument which is the pytorchjob.| Optional|

### Return type
object


## wait_for_condition
> wait_for_condition(name,
>                    expected_condition,
>                    namespace=None,
>                    timeout_seconds=600,
>                    polling_interval=30,
>                    status_callback=None):


Waits until any of the specified conditions occur.

### Example

```python
from kubeflow.training import PyTorchJobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.wait_for_condition('mnist', expected_condition=["Succeeded", "Failed"], namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The PyTorchJob name.| |
expected_condition  |List |A list of conditions. Function waits until any of the supplied conditions is reached.| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace. | Optional|
timeout_seconds | int | How long to wait for the job, default wait for 600 seconds. | Optional|
polling_interval | int | How often to poll for the status of the job.| Optional|
status_callback | str | Callable. If supplied this callable is invoked after we poll the job. Callable takes a single argument which is the pytorchjob.| Optional|

### Return type
object

## get_job_status
> get_job_status(name, namespace=None)

Returns PyTorchJob status, such as Running, Failed or Succeeded.

### Example

```python
from kubeflow.training import PyTorchJobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.get_job_status('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The PyTorchJob name. | |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace.| Optional |

### Return type
Str

## is_job_running
> is_job_running(name, namespace=None)

Returns True if the PyTorchJob running; false otherwise.

### Example

```python
from kubeflow.training import PyTorchJobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.is_job_running('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The PyTorchJob name.| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace.| Optional |

### Return type
Bool

## is_job_succeeded
> is_job_succeeded(name, namespace=None)

Returns True if the PyTorchJob succeeded; false otherwise.

### Example

```python
from kubeflow.training import PyTorchJobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.is_job_succeeded('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The PyTorchJob name.| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace.| Optional |

### Return type
Bool

## get_pod_names
> get_pod_names(name, namespace=None, master=False, replica_type=None, replica_index=None)

Get pod names of the PyTorchJob.

### Example

```python
from kubeflow.training import PyTorchJobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.get_pod_names('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The PyTorchJob name.| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace.| Optional |
master  | bool | Only get pod with label 'job-role: master' pod if True. | |
replica_type | str | User can specify one of 'master, worker' to only get one type pods. By default get all type pods.| |
replica_index | str | User can specfy replica index to get one pod of the PyTorchJob. | |

### Return type
Set


## get_logs
> get_logs(name, namespace=None, master=True, replica_type=None, replica_index=None, follow=False)

Get training logs of the PyTorchJob. By default only get the logs of Pod that has labels 'job-role: master', to get all pods logs, specfy the `master=False`.

### Example

```python
from kubeflow.training import PyTorchJobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.get_logs('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The PyTorchJob name.| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace.| Optional |
master  | bool | Only get pod with label 'job-role: master' pod if True. | |
replica_type  | str | User can specify one of 'master, worker' to only get one type pods. By default get all type pods.| |
replica_index | str | User can specfy replica index to get one pod of the PyTorchJob. | |
follow | bool | Follow the log stream of the pod. Defaults to false. | |

### Return type
Str
