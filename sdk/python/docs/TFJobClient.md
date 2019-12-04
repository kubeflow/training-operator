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


## create
> create(tfjob, namespace=None)

Create the provided tfjob in the specified namespace

### Example

```python
from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container

from kubeflow.tfjob import constants
from kubeflow.tfjob import utils
from kubeflow.tfjob import V1ReplicaSpec
from kubeflow.tfjob import V1TFJob
from kubeflow.tfjob import V1TFJobList
from kubeflow.tfjob import V1TFJobSpec
from kubeflow.tfjob import TFJobClient


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

tfjob = V1TFJob(
    api_version="kubeflow.org/v1",
    kind="TFJob",
    metadata=V1ObjectMeta(name="mnist",namespace=namespace),
    spec=V1TFJobSpec(
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
tfjob  | [V1TFJob](V1TFJob.md) | tfjob defination| Required |
namespace | str | Namespace for tfjob deploying to. If the `namespace` is not defined, will align with tfjob definition, or use current or default namespace if namespace is not specified in tfjob definition.  | Optional |

### Return type
object

## get
> get(name=None, namespace=None)

Get the created tfjob in the specified namespace

### Example

```python
from kubeflow.tfjob import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.get('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | tfjob name. If the `name` is not specified, it will get all tfjobs in the namespace.| Optional. |
namespace | str | The tfjob's namespace. Defaults to current or default namespace.| Optional |


### Return type
object


## patch
> patch(name, tfjob, namespace=None)

Patch the created tfjob in the specified namespace.

Note that if you want to set the field from existing value to `None`, `patch` API may not work, you need to use [replace](#replace) API to remove the field value.

### Example

```python

tfjob = V1TFJob(
    api_version="kubeflow.org/v1",
    ... #update something in TFJob spec
)

tfjob_client = TFJobClient()
tfjob_client.patch('mnist', isvc)

```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
tfjob  | [V1TFob](V1TFJob.md) | tfjob defination| Required |
namespace | str | The tfjob's namespace for patching. If the `namespace` is not defined, will align with tfjob definition, or use current or default namespace if namespace is not specified in tfjob definition. | Optional|

### Return type
object


## delete
> delete(name, namespace=None)

Delete the created tfjob in the specified namespace

### Example

```python
from kubeflow.tfjob import TFJobClient

tfjob_client = TFJobClient()
tfjob_client.delete('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | tfjob name| |
namespace | str | The tfjob's namespace. Defaults to current or default namespace. | Optional|

### Return type
object
