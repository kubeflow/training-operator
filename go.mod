module github.com/kubeflow/tf-operator

go 1.13

require (
	github.com/go-openapi/spec v0.19.7
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.5
	github.com/kubeflow/common v0.0.0-20200327002023-0b3a4c3fca85
	github.com/kubernetes-sigs/kube-batch v0.5.0
	github.com/onrik/logrus v0.5.1
	github.com/prometheus/client_golang v1.5.1
	github.com/sirupsen/logrus v1.5.0
	github.com/stretchr/testify v1.5.1
	github.com/tidwall/gjson v1.6.0 // indirect
	k8s.io/api v0.16.8
	k8s.io/apimachinery v0.16.8
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/code-generator v0.16.8
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/kubernetes v1.16.8
)

replace (
	github.com/kubernetes-sigs/kube-batch => github.com/kubernetes-sigs/kube-batch v0.0.0-20200402050024-a7a29c3ad496
	k8s.io/api => k8s.io/api v0.16.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.8
	k8s.io/apiserver => k8s.io/apiserver v0.16.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.8
	k8s.io/client-go => k8s.io/client-go v0.16.8
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.8
	k8s.io/code-generator => k8s.io/code-generator v0.16.8
	k8s.io/component-base => k8s.io/component-base v0.16.8
	k8s.io/cri-api => k8s.io/cri-api v0.16.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.8
	k8s.io/kubectl => k8s.io/kubectl v0.16.8
	k8s.io/kubelet => k8s.io/kubelet v0.16.8
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.16.8
	k8s.io/metrics => k8s.io/metrics v0.16.8
	k8s.io/node-api => k8s.io/node-api v0.16.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.16.8
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.16.8
	k8s.io/sample-controller => k8s.io/sample-controller v0.16.8
)
