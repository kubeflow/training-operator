# KubeflowOrgV1TFJobSpec

TFJobSpec is a desired state description of the TFJob.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enable_dynamic_worker** | **bool** | A switch to enable dynamic worker | [optional] 
**run_policy** | [**V1RunPolicy**](V1RunPolicy.md) |  | 
**success_policy** | **str** | SuccessPolicy defines the policy to mark the TFJob as succeeded. Default to \&quot;\&quot;, using the default rules. | [optional] 
**tf_replica_specs** | [**dict(str, V1ReplicaSpec)**](V1ReplicaSpec.md) | A map of TFReplicaType (type) to ReplicaSpec (value). Specifies the TF cluster configuration. For example,   {     \&quot;PS\&quot;: ReplicaSpec,     \&quot;Worker\&quot;: ReplicaSpec,   } | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


