# V1TFJobSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active_deadline_seconds** | **int** | Specifies the duration (in seconds) since startTime during which the job can remain active before it is terminated. Must be a positive integer. This setting applies only to pods where restartPolicy is OnFailure or Always. | [optional] 
**backoff_limit** | **int** | Number of retries before marking this job as failed. | [optional] 
**clean_pod_policy** | **str** | Defines the policy for cleaning up pods after the TFJob completes. Defaults to Running. | [optional] 
**tf_replica_specs** | [**dict(str, V1ReplicaSpec)**](V1ReplicaSpec.md) | A map of TFReplicaType (type) to ReplicaSpec (value). Specifies the TF cluster configuration. For example,   {     \&quot;PS\&quot;: ReplicaSpec,     \&quot;Worker\&quot;: ReplicaSpec,   } | 
**ttl_seconds_after_finished** | **int** | Defines the TTL for cleaning up finished TFJobs (temporary before kubernetes adds the cleanup controller). It may take extra ReconcilePeriod seconds for the cleanup, since reconcile gets called periodically. Defaults to infinite. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


