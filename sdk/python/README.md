# Kubeflow TFJob SDK
Python SDK for TF-Operator

## Requirements.

Python 2.7 and 3.5+

## Installation & Usage
### pip install

```sh
pip install kubeflow-tfjob
```

Then import the package:
```python
from kubeflow import tfjob 
```

### Setuptools

Install via [Setuptools](http://pypi.python.org/pypi/setuptools).

```sh
python setup.py install --user
```
(or `sudo python setup.py install` to install the package for all users)


## Getting Started

Please follow the [sample](examples/kubeflow-tfjob-sdk.ipynb) to create, update and delete TFJob.

## Documentation for API Endpoints

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

## Documentation For Models

 - [V1JobCondition](docs/V1JobCondition.md)
 - [V1JobStatus](docs/V1JobStatus.md)
 - [V1ReplicaSpec](docs/V1ReplicaSpec.md)
 - [V1ReplicaStatus](docs/V1ReplicaStatus.md)
 - [V1TFJob](docs/V1TFJob.md)
 - [V1TFJobList](docs/V1TFJobList.md)
 - [V1TFJobSpec](docs/V1TFJobSpec.md)

