# KubeflowOrgV1PyTorchJobSpec

PyTorchJobSpec is a desired state description of the PyTorchJob.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**elastic_policy** | [**KubeflowOrgV1ElasticPolicy**](KubeflowOrgV1ElasticPolicy.md) |  | [optional] 
**pytorch_replica_specs** | [**dict(str, V1ReplicaSpec)**](V1ReplicaSpec.md) | A map of PyTorchReplicaType (type) to ReplicaSpec (value). Specifies the PyTorch cluster configuration. For example,   {     \&quot;Master\&quot;: PyTorchReplicaSpec,     \&quot;Worker\&quot;: PyTorchReplicaSpec,   } | 
**run_policy** | [**V1RunPolicy**](V1RunPolicy.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


