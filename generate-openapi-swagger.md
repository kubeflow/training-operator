# How to Generate an OpenAPI Spec for TF Operator

From the root of the repo,
`go run hack/gen-openapi-spec/main.go v0.6 > pkg/apis/tensorflow/v1/swagger.json`

## TODO

Add instructions for creating an sdk from the swagger.json generated.