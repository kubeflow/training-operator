# TFJobClient

> TFJobClient(config_file=None, context=None, client_configuration=None, persist_config=True)

User can loads authentication and cluster information from kube-config file and stores them in kubernetes.client.configuration. Parameters are as following:

parameter |  Description
------------ | -------------
config_file | Name of the kube-config file. Defaults to `~/.kube/config`. Note that for the case that the SDK is running in cluster and you want to operate tfjob in another remote cluster, user must set `config_file` to load kube-config file explicitly, e.g. `TFJobClient(config_file="~/.kube/config")`. |
context |Set the active context. If is set to None, current_context from config file will be used.|
client_configuration | The kubernetes.client.Configuration to set configs to.|
persist_config | If True, config file will be updated when changed (e.g GCP token refresh).|


The APIs for TFJobClient are as following:

Class | Method |  Description
------------ | ------------- | -------------
TFJobClient| [create](#create) | Create TFJob|
TFJobClient | [get](#get)    | Get the specified TFJob or all TFJob in the namespace |
TFJobClient | [patch](#patch)  | Patch the specified TFJob|
TFJobClient | [delete](#delete) | Delete the specified TFJob |
TFJobClient | [wait_for_job](#wait_for_job) | Wait for the specified job to finish |
TFJobClient | [wait_for_condition](#wait_for_condition) | Waits until any of the specified conditions occur |
TFJobClient | [get_job_status](#get_job_status) | Get the TFJob status|
TFJobClient | [is_job_running](#is_job_running) | Check if the TFJob status is running |
TFJobClient | [is_job_succeeded](#is_job_succeeded) | Check if the TFJob status is Succeeded |
TFJobClient | [get_pod_names](#get_pod_names) | Get pod names of TFJob |
TFJobClient | [get_logs](#get_logs) | Get training logs of the TFJob |


## create
> create(tfjob, namespace=None)

Create the provided tfjob in the specified namespace

### Example

```python
from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container

from kubeflow.training import constants
from kubeflow.training import utils
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1TFJob
from kubeflow.training import KubeflowOrgV1TFJobList
from kubeflow.training import KubeflowOrgV1TFJobSpec
from kubeflow.training import TFJobClient


container = V1Container(
    name="tensorflow",
    image="gcr.io/kubeflow-ci/tf-mnist-with-summaries:1.0",
    command=[
        "python",
        "/var/tf_mnist/mnist_with_summaries.py",
        "--log_dir=/train/logs", "--learning_rate=0.01",
        "--batch_size=150"
        ]
)

worker = V1ReplicaSpec(
    replicas=1,
    restart_policy="Never",
    template=V1PodTemplateSpec(
        spec=V1PodSpec(
            containers=[container]
        )
    )
)

tfjob = KubeflowOrgV1TFJob(
    api_version="kubeflow.org/v1",
    kind="TFJob",
    metadata=V1ObjectMeta(name="mnist",namespace=namespace),
    spec=KubeflowOrgV1TFJobSpec(
        clean_pod_policy="None",
        tf_replica_specs={"Worker": worker}
    )
)


tfjob_client = TFJobClient()
tfjob_client.create(tfjob)

```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
tfjob  | [KubeflowOrgV1TFJob](KubeflowOrgV1TFJob.md) | tfjob defination| Required |
namespace | str | Namespace for tfjob deploying to. If the `namespace` is not defined, will align with tfjob definition, or use current or default namespace if namespace is not specified in tfjob definition.  | Optional |

### Return type
object

## get
> get(name=None, namespace=None, watch=False, timeout_seconds=600)

Get the created tfjob in the specified namespace

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.get('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name. If the `name` is not specified, it will get all tfjobs in the namespace.| Optional. |
namespace | str | The tfjob's namespace. Defaults to current or default namespace.| Optional |
watch | bool | Watch the created TFJob if `True`, otherwise will return the created TFJob object. Stop watching if TFJob reaches the optional specified `timeout_seconds` or once the TFJob status `Succeeded` or `Failed`. | Optional |
timeout_seconds | int | Timeout seconds for watching. Defaults to 600. | Optional |

### Return type
object


## patch
> patch(name, tfjob, namespace=None)

Patch the created tfjob in the specified namespace.

Note that if you want to set the field from existing value to `None`, `patch` API may not work, you need to use [replace](#replace) API to remove the field value.

### Example

```python

tfjob = KubeflowOrgV1TFJob(
    api_version="kubeflow.org/v1",
    ... #update something in TFJob spec
)

tfjob_client = TFJobClient()
tfjob_client.patch('mnist', isvc)

```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
tfjob  | [KubeflowOrgV1TFJob](KubeflowOrgV1TFJob.md) | tfjob defination| Required |
namespace | str | The tfjob's namespace for patching. If the `namespace` is not defined, will align with tfjob definition, or use current or default namespace if namespace is not specified in tfjob definition. | Optional|

### Return type
object


## delete
> delete(name, namespace=None)

Delete the created tfjob in the specified namespace

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.delete('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name.| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace. | Optional|

### Return type
object


## wait_for_job
> wait_for_job(name,
>              namespace=None,
>              timeout_seconds=600,
>              polling_interval=30,
>              watch=False,
>              status_callback=None):

Wait for the specified job to finish.

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.wait_for_job('mnist', namespace='kubeflow')

# The API also supports watching the TFJob status till it's Succeeded or Failed.
tfjob_client.wait_for_job('mnist', namespace=namespace, watch=True)
NAME                           STATE                TIME
mnist                          Created              2019-12-31T09:20:07Z
mnist                          Running              2019-12-31T09:20:19Z
mnist                          Running              2019-12-31T09:20:19Z
mnist                          Succeeded            2019-12-31T09:22:04Z
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name.| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace. | Optional|
timeout_seconds | int | How long to wait for the job, default wait for 600 seconds. | Optional|
polling_interval | int | How often to poll for the status of the job.| Optional|
status_callback | str | Callable. If supplied this callable is invoked after we poll the job. Callable takes a single argument which is the tfjob.| Optional|
watch | bool | Watch the TFJob if `True`. Stop watching if TFJob reaches the optional specified `timeout_seconds` or once the TFJob status `Succeeded` or `Failed`. | Optional |

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
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.wait_for_condition('mnist', expected_condition=["Succeeded", "Failed"], namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name.| |
expected_condition  |List |A list of conditions. Function waits until any of the supplied conditions is reached.| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace. | Optional|
timeout_seconds | int | How long to wait for the job, default wait for 600 seconds. | Optional|
polling_interval | int | How often to poll for the status of the job.| Optional|
status_callback | str | Callable. If supplied this callable is invoked after we poll the job. Callable takes a single argument which is the tfjob.| Optional|

### Return type
object

## get_job_status
> get_job_status(name, namespace=None)

Returns TFJob status, such as Running, Failed or Succeeded.

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.get_job_status('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name. | |
namespace | str | The tfjob's namespace. Defaults to current or default namespace.| Optional |

### Return type
Str

## is_job_running
> is_job_running(name, namespace=None)

Returns True if the TFJob running; false otherwise.

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.is_job_running('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name.| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace.| Optional |

### Return type
Bool

## is_job_succeeded
> is_job_succeeded(name, namespace=None)

Returns True if the TFJob succeeded; false otherwise.

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.is_job_succeeded('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name.| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace.| Optional |

### Return type
Bool


## get_pod_names
> get_pod_names(name, namespace=None, master=False, replica_type=None, replica_index=None)

Get pod names of the TFJob.

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.get_pod_names('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name.| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace.| Optional |
master  | bool | Only get pod with label 'job-role: master' pod if True. | |
replica_type | str | User can specify one of 'worker, ps, chief' to only get one type pods. By default get all type pods.| |
replica_index | str | User can specfy replica index to get one pod of the TFJob. | |

### Return type
Set


## get_logs
> get_logs(name, namespace=None, master=True, replica_type=None, replica_index=None, follow=False)

Get training logs of the TFJob. By default only get the logs of Pod that has labels 'job-role: master', to get all pods logs, specfy the `master=False`.

### Example

```python
from kubeflow.training import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.get_logs('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | The TFJob name.| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace.| Optional |
master  | bool | Only get pod with label 'job-role: master' pod if True. | |
replica_type  | str | User can specify one of 'worker, ps, chief' to only get one type pods. By default get all type pods.| |
replica_index | str | User can specfy replica index to get one pod of the TFJob. | |
follow | bool | Follow the log stream of the pod. Defaults to false. | |

### Return type
Str
