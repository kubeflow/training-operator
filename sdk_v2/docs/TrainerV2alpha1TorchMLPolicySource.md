# TrainerV2alpha1TorchMLPolicySource

TorchMLPolicySource represents a PyTorch runtime configuration.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**elastic_policy** | [**TrainerV2alpha1TorchElasticPolicy**](TrainerV2alpha1TorchElasticPolicy.md) |  | [optional] 
**num_proc_per_node** | **str** | Number of processes per node. This value is inserted into the &#x60;--nproc-per-node&#x60; argument of the &#x60;torchrun&#x60; CLI. Supported values: &#x60;auto&#x60;, &#x60;cpu&#x60;, &#x60;gpu&#x60;, or int value. Defaults to &#x60;auto&#x60;. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


