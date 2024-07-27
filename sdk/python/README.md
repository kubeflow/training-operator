# Kubeflow Training SDK

Python SDK for Training Operator

## Requirements.

Python >= 3.8

Training Python SDK follows [Python release cycle](https://devguide.python.org/versions/#python-release-cycle)
for supported Python versions.

## Installation & Usage

### pip install

```sh
pip install kubeflow-training
```

Then import the package:

```python
from kubeflow import training
```

### Setuptools

Install via [Setuptools](http://pypi.python.org/pypi/setuptools).

```sh
python setup.py install --user
```

(or `sudo python setup.py install` to install the package for all users)

## Getting Started

Please follow the [Getting Started guide](https://www.kubeflow.org/docs/components/training/overview/#getting-started)
or check Training Operator [examples](../../examples).

## Documentation for API Endpoints

TODO(andreyvelich): These docs are outdated. Please track this issue for the status:
https://github.com/kubeflow/katib/issues/2081

Class | Method | Description
------------ | -------------  | -------------
[TFJobClient](docs/TFJobClient.md) | [create](docs/TFJobClient.md#create) | Create TFJob|
[TFJobClient](docs/TFJobClient.md) | [get](docs/TFJobClient.md#get)    | Get or watch the specified TFJob or all TFJob in the namespace |
[TFJobClient](docs/TFJobClient.md) | [patch](docs/TFJobClient.md#patch)  | Patch the specified TFJob|
[TFJobClient](docs/TFJobClient.md) | [delete](docs/TFJobClient.md#delete) | Delete the specified TFJob |
[TFJobClient](docs/TFJobClient.md) | [wait_for_job](docs/TFJobClient.md#wait_for_job) | Wait for the specified job to finish |
[TFJobClient](docs/TFJobClient.md) | [wait_for_condition](docs/TFJobClient.md#wait_for_condition) | Waits until any of the specified conditions occur |
[TFJobClient](docs/TFJobClient.md) | [get_job_status](docs/TFJobClient.md#get_job_status) | Get the TFJob status|
[TFJobClient](docs/TFJobClient.md) | [is_job_running](docs/TFJobClient.md#is_job_running) | Check if the TFJob status is Running |
[TFJobClient](docs/TFJobClient.md) | [is_job_succeeded](docs/TFJobClient.md#is_job_succeeded) | Check if the TFJob status is Succeeded |
[TFJobClient](docs/TFJobClient.md) | [get_pod_names](docs/TFJobClient.md#get_pod_names) | Get pod names of TFJob |
[TFJobClient](docs/TFJobClient.md) | [get_logs](docs/TFJobClient.md#get_logs) | Get training logs of the TFJob |
[PyTorchJobClient](docs/PyTorchJobClient.md) | [create](docs/PyTorchJobClient.md#create) | Create PyTorchJob|
[PyTorchJobClient](docs/PyTorchJobClient.md) | [get](docs/PyTorchJobClient.md#get)    | Get the specified PyTorchJob or all PyTorchJob in the namespace |
[PyTorchJobClient](docs/PyTorchJobClient.md) | [patch](docs/PyTorchJobClient.md#patch)  | Patch the specified PyTorchJob|
[PyTorchJobClient](docs/PyTorchJobClient.md) | [delete](docs/PyTorchJobClient.md#delete) | Delete the specified PyTorchJob |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [wait_for_job](docs/PyTorchJobClient.md#wait_for_job) | Wait for the specified job to finish |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [wait_for_condition](docs/PyTorchJobClient.md#wait_for_condition) | Waits until any of the specified conditions occur |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [get_job_status](docs/PyTorchJobClient.md#get_job_status) | Get the PyTorchJob status|
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [is_job_running](docs/PyTorchJobClient.md#is_job_running) | Check if the PyTorchJob running |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [is_job_succeeded](docs/PyTorchJobClient.md#is_job_succeeded) | Check if the PyTorchJob Succeeded |
[PyTorchJobClient](docs/PyTorchJobClient.md) | [get_pod_names](docs/PyTorchJobClient.md#get_pod_names) | Get pod names of PyTorchJob |
[PyTorchJobClient](docs/PyTorchJobClient.md)| [get_logs](docs/PyTorchJobClient.md#get_logs) | Get training logs of the PyTorchJob |

## Documentation For Models

- [V1JobCondition](docs/V1JobCondition.md)
- [V1JobStatus](docs/V1JobStatus.md)
- [V1PyTorchJob](docs/KubeflowOrgV1PyTorchJob.md)
- [V1PyTorchJobList](docs/KubeflowOrgV1PyTorchJobList.md)
- [V1PyTorchJobSpec](docs/KubeflowOrgV1PyTorchJobSpec.md)
- [V1ReplicaSpec](docs/V1ReplicaSpec.md)
- [V1ReplicaStatus](docs/V1ReplicaStatus.md)
- [V1RunPolicy](docs/V1RunPolicy.md)
- [V1SchedulingPolicy](docs/V1SchedulingPolicy.md)
- [V1TFJob](docs/KubeflowOrgV1TFJob.md)
- [V1TFJobList](docs/KubeflowOrgV1TFJobList.md)
- [V1TFJobSpec](docs/KubeflowOrgV1TFJobSpec.md)
- [V1XGBoostJob](docs/KubeflowOrgV1XGBoostJob.md)
- [V1XGBoostJobList](docs/KubeflowOrgV1XGBoostJobList.md)
- [V1XGBoostJobSpec](docs/KubeflowOrgV1XGBoostJobSpec.md)

## Building conformance tests

Run

```
docker build . -f Dockerfile.conformance -t <tag>
```
