## Test Job Controller

This is a Test Job Controller example. As you can see, we have job crd definition under `apis/test_job/v1`. 
[code-generator](https://github.com/kubernetes/code-generator) generate deepcopy, clientset and other libraries.

`controler.v1/test_job/test_job_controller` defines a struct `TestJobController` which implements [commonv1.ControllerInterface](../pkg/apis/common/v1/interface.go) 

```yaml
├── README.md
├── apis
│   └── test_job
│       └── v1
│           ├── constants.go
│           ├── defaults.go
│           ├── doc.go
│           ├── openapi_generated.go
│           ├── register.go
│           ├── types.go
│           ├── zz_generated.deepcopy.go
│           └── zz_generated.defaults.go
├── client
│   ├── clientset
│   ├── informers
│   └── listers
├── controller.v1
│   └── test_job
│       └── test_job_controller.go
└── test_util
```