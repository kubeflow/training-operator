# KubeflowOrgV1PaddleElasticPolicy

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**max_replicas** | **int** | upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas, defaults to null. | [optional] 
**max_restarts** | **int** | MaxRestarts is the limit for restart times of pods in elastic mode. | [optional] 
**metrics** | [**list[K8sIoApiAutoscalingV2beta2MetricSpec]**](K8sIoApiAutoscalingV2beta2MetricSpec.md) | Metrics contains the specifications which are used to calculate the desired replica count (the maximum replica count across all metrics will be used).  The desired replica count is calculated with multiplying the ratio between the target value and the current value by the current number of pods. Ergo, metrics used must decrease as the pod count is increased, and vice-versa.  See the individual metric source types for more information about how each type of metric must respond. If not set, the HPA will not be created. | [optional] 
**min_replicas** | **int** | minReplicas is the lower limit for the number of replicas to which the training job can scale down.  It defaults to null. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


