# V1TableRowCondition

TableRowCondition allows a row to be marked with additional information.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** | Human readable message indicating details about last transition. | [optional] 
**reason** | **str** | (brief) machine readable reason for the condition&#39;s last transition. | [optional] 
**status** | **str** | Status of the condition, one of True, False, Unknown. | [default to '']
**type** | **str** | Type of row condition. The only defined value is &#39;Completed&#39; indicating that the object this row represents has reached a completed state and may be given less visual priority than other rows. Clients are not required to honor any conditions but should be consistent where possible about handling the conditions. | [default to '']

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


