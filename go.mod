module github.com/kubeflow/tf-operator

go 1.14

require (
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/spec v0.20.3
	github.com/kubeflow/common v0.3.4
	github.com/onrik/logrus v0.2.2-0.20181225141908-a09d5cdcdc62
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.6.0
	k8s.io/api v0.19.9
	k8s.io/apiextensions-apiserver v0.19.9
	k8s.io/apimachinery v0.19.9
	k8s.io/client-go v0.19.9
	k8s.io/code-generator v0.19.9
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	sigs.k8s.io/controller-runtime v0.7.2
	volcano.sh/apis v1.2.0-k8s1.19.6
)
