# KubeflowOrgV1JAXJobSpec

JAXJobSpec is a desired state description of the JAXJob.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**jax_replica_specs** | [**dict(str, KubeflowOrgV1ReplicaSpec)**](KubeflowOrgV1ReplicaSpec.md) | A map of JAXReplicaType (type) to ReplicaSpec (value). Specifies the JAX cluster configuration. For example,   {     \&quot;Worker\&quot;: JAXReplicaSpec,   } | 
**run_policy** | [**KubeflowOrgV1RunPolicy**](KubeflowOrgV1RunPolicy.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


