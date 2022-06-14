# KubeflowOrgV1MPIJobSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**clean_pod_policy** | **str** | CleanPodPolicy defines the policy that whether to kill pods after the job completes. Defaults to None. | [optional] 
**main_container** | **str** | MainContainer specifies name of the main container which executes the MPI code. | [optional] 
**mpi_replica_specs** | [**dict(str, V1ReplicaSpec)**](V1ReplicaSpec.md) | &#x60;MPIReplicaSpecs&#x60; contains maps from &#x60;MPIReplicaType&#x60; to &#x60;ReplicaSpec&#x60; that specify the MPI replicas to run. | 
**run_policy** | [**V1RunPolicy**](V1RunPolicy.md) |  | [optional] 
**slots_per_worker** | **int** | Specifies the number of slots per worker used in hostfile. Defaults to 1. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


