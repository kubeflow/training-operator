# V1JobStatus

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**completion_time** | [**V1Time**](V1Time.md) | Represents time when the job was completed. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC. | [optional] 
**conditions** | [**list[V1JobCondition]**](V1JobCondition.md) | Conditions is an array of current observed job conditions. | 
**last_reconcile_time** | [**V1Time**](V1Time.md) | Represents last time when the job was reconciled. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC. | [optional] 
**replica_statuses** | [**dict(str, V1ReplicaStatus)**](V1ReplicaStatus.md) | ReplicaStatuses is map of ReplicaType and ReplicaStatus, specifies the status of each replica. | 
**start_time** | [**V1Time**](V1Time.md) | Represents time when the job was acknowledged by the job controller. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


