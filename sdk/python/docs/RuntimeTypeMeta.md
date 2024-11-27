# RuntimeTypeMeta

TypeMeta is shared by all top level objects. The proper way to use it is to inline it in your type, like this:   type MyAwesomeAPIObject struct {       runtime.TypeMeta    `json:\",inline\"`       ... // other fields  }  func (obj *MyAwesomeAPIObject) SetGroupVersionKind(gvk *metav1.GroupVersionKind) { metav1.UpdateTypeMeta(obj,gvk) }; GroupVersionKind() *GroupVersionKind  TypeMeta is provided here for convenience. You may use it directly from this package or define your own with the same fields.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** |  | [optional] 
**kind** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


