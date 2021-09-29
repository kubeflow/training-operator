# V1ReplicaSpec

ReplicaSpec is a description of the replica
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**replicas** | **int** | Replicas is the desired number of replicas of the given template. If unspecified, defaults to 1. | [optional] 
**restart_policy** | **str** | Restart policy for all replicas within the job. One of Always, OnFailure, Never and ExitCode. Default to Never. | [optional] 
**template** | [**V1PodTemplateSpec**](V1PodTemplateSpec.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


