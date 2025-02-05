# TrainerV2alpha1TrainJobSpec

TrainJobSpec represents specification of the desired TrainJob.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **dict(str, str)** | Annotations to apply for the derivative JobSet and Jobs. They will be merged with the TrainingRuntime values. | [optional] 
**dataset_config** | [**TrainerV2alpha1DatasetConfig**](TrainerV2alpha1DatasetConfig.md) |  | [optional] 
**labels** | **dict(str, str)** | Labels to apply for the derivative JobSet and Jobs. They will be merged with the TrainingRuntime values. | [optional] 
**managed_by** | **str** | ManagedBy is used to indicate the controller or entity that manages a TrainJob. The value must be either an empty, &#x60;kubeflow.org/trainjob-controller&#x60; or &#x60;kueue.x-k8s.io/multikueue&#x60;. The built-in TrainJob controller reconciles TrainJob which don&#39;t have this field at all or the field value is the reserved string &#x60;kubeflow.org/trainjob-controller&#x60;, but delegates reconciling TrainJobs with a &#39;kueue.x-k8s.io/multikueue&#39; to the Kueue. The field is immutable. Defaults to &#x60;kubeflow.org/trainjob-controller&#x60; | [optional] 
**model_config** | [**TrainerV2alpha1ModelConfig**](TrainerV2alpha1ModelConfig.md) |  | [optional] 
**pod_spec_overrides** | [**list[TrainerV2alpha1PodSpecOverride]**](TrainerV2alpha1PodSpecOverride.md) | Custom overrides for the training runtime. | [optional] 
**runtime_ref** | [**TrainerV2alpha1RuntimeRef**](TrainerV2alpha1RuntimeRef.md) |  | 
**suspend** | **bool** | Whether the controller should suspend the running TrainJob. Defaults to false. | [optional] 
**trainer** | [**TrainerV2alpha1Trainer**](TrainerV2alpha1Trainer.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


