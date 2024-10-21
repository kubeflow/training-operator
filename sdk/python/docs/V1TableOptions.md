# V1TableOptions

TableOptions are used when a Table is requested by the caller.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources | [optional] 
**include_object** | **str** | includeObject decides whether to include each object along with its columnar information. Specifying \&quot;None\&quot; will return no object, specifying \&quot;Object\&quot; will return the full object contents, and specifying \&quot;Metadata\&quot; (the default) will return the object&#39;s metadata in the PartialObjectMetadata kind in version v1beta1 of the meta.k8s.io API group. | [optional] 
**kind** | **str** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


