# V1RunPolicy

RunPolicy encapsulates various runtime policies of the distributed training job, for example how to clean up resources and how long the job can stay active.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active_deadline_seconds** | **int** | Specifies the duration in seconds relative to the startTime that the job may be active before the system tries to terminate it; value must be positive integer. | [optional] 
**backoff_limit** | **int** | Optional number of retries before marking this job failed. | [optional] 
**clean_pod_policy** | **str** | CleanPodPolicy defines the policy to kill pods after the job completes. Default to Running. | [optional] 
**scheduling_policy** | [**V1SchedulingPolicy**](V1SchedulingPolicy.md) |  | [optional] 
**ttl_seconds_after_finished** | **int** | TTLSecondsAfterFinished is the TTL to clean up jobs. It may take extra ReconcilePeriod seconds for the cleanup, since reconcile gets called periodically. Default to infinite. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


