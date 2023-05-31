# KubeflowOrgV1PaddleJobSpec

PaddleJobSpec is a desired state description of the PaddleJob.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**elastic_policy** | [**KubeflowOrgV1PaddleElasticPolicy**](KubeflowOrgV1PaddleElasticPolicy.md) |  | [optional] 
**paddle_replica_specs** | [**dict(str, KubeflowOrgV1ReplicaSpec)**](KubeflowOrgV1ReplicaSpec.md) | A map of PaddleReplicaType (type) to ReplicaSpec (value). Specifies the Paddle cluster configuration. For example,   {     \&quot;Master\&quot;: PaddleReplicaSpec,     \&quot;Worker\&quot;: PaddleReplicaSpec,   } | 
**run_policy** | [**KubeflowOrgV1RunPolicy**](KubeflowOrgV1RunPolicy.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


