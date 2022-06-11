# KubeflowOrgV1MXJobSpec

MXJobSpec defines the desired state of MXJob
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_mode** | **str** | JobMode specify the kind of MXjob to do. Different mode may have different MXReplicaSpecs request | [default to '']
**mx_replica_specs** | [**dict(str, V1ReplicaSpec)**](V1ReplicaSpec.md) | MXReplicaSpecs is map of commonv1.ReplicaType and commonv1.ReplicaSpec specifies the MX replicas to run. For example,   {     \&quot;Scheduler\&quot;: commonv1.ReplicaSpec,     \&quot;Server\&quot;: commonv1.ReplicaSpec,     \&quot;Worker\&quot;: commonv1.ReplicaSpec,   } | 
**run_policy** | [**V1RunPolicy**](V1RunPolicy.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


