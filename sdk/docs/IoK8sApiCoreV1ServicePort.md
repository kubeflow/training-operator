# IoK8sApiCoreV1ServicePort

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | The name of this port within the service. This must be a DNS_LABEL. All ports within a ServiceSpec must have unique names. This maps to the &#x27;Name&#x27; field in EndpointPort objects. Optional if only one ServicePort is defined on this service. | [optional] 
**node_port** | **int** | The port on each node on which this service is exposed when type&#x3D;NodePort or LoadBalancer. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the ServiceType of this Service requires one. More info: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport | [optional] 
**port** | **int** | The port that will be exposed by this service. | 
**protocol** | **str** | The IP protocol for this port. Supports \&quot;TCP\&quot; and \&quot;UDP\&quot;. Default is TCP. | [optional] 
**target_port** | [**IoK8sApimachineryPkgUtilIntstrIntOrString**](IoK8sApimachineryPkgUtilIntstrIntOrString.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

