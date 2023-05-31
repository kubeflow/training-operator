# KubeflowOrgV1JobStatus

JobStatus represents the current observed state of the training Job.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**completion_time** | [**datetime**](V1Time.md) |  | [optional] 
**conditions** | [**list[KubeflowOrgV1JobCondition]**](KubeflowOrgV1JobCondition.md) | Conditions is an array of current observed job conditions. | 
**last_reconcile_time** | [**datetime**](V1Time.md) |  | [optional] 
**replica_statuses** | [**dict(str, KubeflowOrgV1ReplicaStatus)**](KubeflowOrgV1ReplicaStatus.md) | ReplicaStatuses is map of ReplicaType and ReplicaStatus, specifies the status of each replica. | 
**start_time** | [**datetime**](V1Time.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


