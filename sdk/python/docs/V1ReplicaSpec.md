# V1ReplicaSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**replicas** | **int** | Replicas is the desired number of replicas of the given template. If unspecified, defaults to 1. | [optional] 
**restart_policy** | **str** | Restart policy for all replicas within the job. One of Always, OnFailure, Never and ExitCode. Default to Never. | [optional] 
**template** | [**V1PodTemplateSpec**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1PodTemplateSpec.md) | Template is the object that describes the pod that will be created for this replica. RestartPolicy in PodTemplateSpec will be overide by RestartPolicy in ReplicaSpec | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


