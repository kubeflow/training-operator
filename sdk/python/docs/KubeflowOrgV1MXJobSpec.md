# KubeflowOrgV1MXJobSpec

MXJobSpec defines the desired state of MXJob
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_mode** | **str** | JobMode specify the kind of MXjob to do. Different mode may have different MXReplicaSpecs request | [default to '']
**mx_replica_specs** | [**dict(str, KubeflowOrgV1ReplicaSpec)**](KubeflowOrgV1ReplicaSpec.md) | MXReplicaSpecs is map of ReplicaType and ReplicaSpec specifies the MX replicas to run. For example,   {     \&quot;Scheduler\&quot;: ReplicaSpec,     \&quot;Server\&quot;: ReplicaSpec,     \&quot;Worker\&quot;: ReplicaSpec,   } | 
**run_policy** | [**KubeflowOrgV1RunPolicy**](KubeflowOrgV1RunPolicy.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


