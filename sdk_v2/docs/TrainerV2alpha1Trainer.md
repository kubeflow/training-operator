# TrainerV2alpha1Trainer

Trainer represents the desired trainer configuration. Every training runtime contains `trainer` container which represents Trainer.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**args** | **list[str]** | Arguments to the entrypoint for the training container. | [optional] 
**command** | **list[str]** | Entrypoint commands for the training container. | [optional] 
**env** | [**list[V1EnvVar]**](V1EnvVar.md) | List of environment variables to set in the training container. These values will be merged with the TrainingRuntime&#39;s trainer environments. | [optional] 
**image** | **str** | Docker image for the training container. | [optional] 
**num_nodes** | **int** | Number of training nodes. | [optional] 
**num_proc_per_node** | **str** | Number of processes/workers/slots on every training node. For the Torch runtime: &#x60;auto&#x60;, &#x60;cpu&#x60;, &#x60;gpu&#x60;, or int value can be set. For the MPI runtime only int value can be set. | [optional] 
**resources_per_node** | [**V1ResourceRequirements**](V1ResourceRequirements.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


