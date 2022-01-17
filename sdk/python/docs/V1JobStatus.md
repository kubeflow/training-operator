# V1JobStatus

JobStatus represents the current observed state of the training Job.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**completion_time** | [**V1Time**](V1Time.md) |  | [optional] 
**conditions** | [**list[V1JobCondition]**](V1JobCondition.md) | Conditions is an array of current observed job conditions. | 
**last_reconcile_time** | [**V1Time**](V1Time.md) |  | [optional] 
**replica_statuses** | [**dict(str, V1ReplicaStatus)**](V1ReplicaStatus.md) | ReplicaStatuses is map of ReplicaType and ReplicaStatus, specifies the status of each replica. | 
**start_time** | [**V1Time**](V1Time.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


