# V1MXJobSpec

MXJobSpec defines the desired state of MXJob
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_mode** | **str** | JobMode specify the kind of MXjob to do. Different mode may have different MXReplicaSpecs request | 
**mx_replica_specs** | [**dict(str, V1ReplicaSpec)**](V1ReplicaSpec.md) | MXReplicaSpecs is map of common.ReplicaType and common.ReplicaSpec specifies the MX replicas to run. For example,   {     \&quot;Scheduler\&quot;: common.ReplicaSpec,     \&quot;Server\&quot;: common.ReplicaSpec,     \&quot;Worker\&quot;: common.ReplicaSpec,   } | 
**run_policy** | [**V1RunPolicy**](V1RunPolicy.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


