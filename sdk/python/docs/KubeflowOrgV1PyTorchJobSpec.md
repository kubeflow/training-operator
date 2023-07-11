# KubeflowOrgV1PyTorchJobSpec

PyTorchJobSpec is a desired state description of the PyTorchJob.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**elastic_policy** | [**KubeflowOrgV1ElasticPolicy**](KubeflowOrgV1ElasticPolicy.md) |  | [optional] 
**nproc_per_node** | **str** | Number of workers per node; supported values: [auto, cpu, gpu, int]. For more, https://github.com/pytorch/pytorch/blob/26f7f470df64d90e092081e39507e4ac751f55d6/torch/distributed/run.py#L629-L658. Defaults to auto. | [optional] 
**pytorch_replica_specs** | [**dict(str, KubeflowOrgV1ReplicaSpec)**](KubeflowOrgV1ReplicaSpec.md) | A map of PyTorchReplicaType (type) to ReplicaSpec (value). Specifies the PyTorch cluster configuration. For example,   {     \&quot;Master\&quot;: PyTorchReplicaSpec,     \&quot;Worker\&quot;: PyTorchReplicaSpec,   } | 
**run_policy** | [**KubeflowOrgV1RunPolicy**](KubeflowOrgV1RunPolicy.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


