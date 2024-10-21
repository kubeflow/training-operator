# V1Timestamp

Timestamp is a struct that is equivalent to Time, but intended for protobuf marshalling/unmarshalling. It is generated into a serialization that matches Time. Do not use in Go structs.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**nanos** | **int** | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. | [default to 0]
**seconds** | **int** | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. | [default to 0]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


