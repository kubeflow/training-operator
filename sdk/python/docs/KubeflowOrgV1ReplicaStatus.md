# KubeflowOrgV1ReplicaStatus

ReplicaStatus represents the current observed state of the replica.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | **int** | The number of actively running pods. | [optional] 
**failed** | **int** | The number of pods which reached phase Failed. | [optional] 
**label_selector** | [**V1LabelSelector**](V1LabelSelector.md) |  | [optional] 
**selector** | **str** | A Selector is a label query over a set of resources. The result of matchLabels and matchExpressions are ANDed. An empty Selector matches all objects. A null Selector matches no objects. | [optional] 
**succeeded** | **int** | The number of pods which reached phase Succeeded. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


