# KubeflowOrgV1JobStatus

JobStatus represents the current observed state of the training Job.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**completion_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**conditions** | [**list[KubeflowOrgV1JobCondition]**](KubeflowOrgV1JobCondition.md) | Conditions is an array of current observed job conditions. | [optional] 
**last_reconcile_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**replica_statuses** | [**dict(str, KubeflowOrgV1ReplicaStatus)**](KubeflowOrgV1ReplicaStatus.md) | ReplicaStatuses is map of ReplicaType and ReplicaStatus, specifies the status of each replica. | [optional] 
**start_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


