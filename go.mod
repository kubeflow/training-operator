module github.com/kubeflow/tf-operator

go 1.14

require (
	github.com/go-openapi/spec v0.19.2
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.4.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.5.0 // indirect
	github.com/kubeflow/common v0.3.1
	github.com/kubernetes-sigs/kube-batch v0.0.0-20200414051246-2e934d1c8860
	github.com/onrik/logrus v0.2.2-0.20181225141908-a09d5cdcdc62
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.16.9
	k8s.io/apiextensions-apiserver v0.16.9 // indirect
	k8s.io/apimachinery v0.16.10-beta.0
	k8s.io/client-go v0.16.9
	k8s.io/code-generator v0.16.9
	k8s.io/component-base v0.18.2 // indirect
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200410163147-594e756bea31
	k8s.io/kubernetes v1.16.9
	sigs.k8s.io/yaml v1.2.0 // indirect
	volcano.sh/volcano v0.4.0
)

replace (
	k8s.io/api => k8s.io/api v0.16.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.10-beta.0
	k8s.io/apiserver => k8s.io/apiserver v0.16.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.9
	k8s.io/client-go => k8s.io/client-go v0.16.9
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.9
	k8s.io/code-generator => k8s.io/code-generator v0.16.10-beta.0
	k8s.io/component-base => k8s.io/component-base v0.16.9
	k8s.io/cri-api => k8s.io/cri-api v0.16.10-beta.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.9
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.9
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.9
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.9
	k8s.io/kubectl => k8s.io/kubectl v0.16.9
	k8s.io/kubelet => k8s.io/kubelet v0.16.9
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.16.9
	k8s.io/metrics => k8s.io/metrics v0.16.9
	k8s.io/node-api => k8s.io/node-api v0.16.9
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.16.9
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.16.9
	k8s.io/sample-controller => k8s.io/sample-controller v0.16.9
)
