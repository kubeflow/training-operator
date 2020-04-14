module github.com/kubeflow/common

go 1.13

require (
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-openapi/spec v0.19.2
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/stretchr/testify v1.3.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.3.0 // indirect
	k8s.io/api v0.15.10
	k8s.io/apimachinery v0.16.9-beta.0
	k8s.io/client-go v0.16.9-beta.0
	k8s.io/code-generator v0.15.10
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/kubernetes v1.16.2
	volcano.sh/volcano v0.0.0-20200408232842-66fcaa599f10
)

replace (
	k8s.io/api => k8s.io/api v0.15.10
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.15.10
	k8s.io/apimachinery => k8s.io/apimachinery v0.15.10
	k8s.io/apiserver => k8s.io/apiserver v0.15.10
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.15.10
	k8s.io/client-go => k8s.io/client-go v0.15.10
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.15.10
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.15.10
	k8s.io/code-generator => k8s.io/code-generator v0.15.10
	k8s.io/component-base => k8s.io/component-base v0.15.10
	k8s.io/cri-api => k8s.io/cri-api v0.15.11-beta.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.15.10
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.15.10
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.15.10
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.15.10
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.15.10
	k8s.io/kubectl => k8s.io/kubectl v0.15.11-beta.0
	k8s.io/kubelet => k8s.io/kubelet v0.15.10
	k8s.io/kubernetes => k8s.io/kubernetes v1.15.10
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.15.10
	k8s.io/metrics => k8s.io/metrics v0.15.10
	k8s.io/node-api => k8s.io/node-api v0.15.10
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.15.10
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.15.10
	k8s.io/sample-controller => k8s.io/sample-controller v0.15.10
)
