package k8sutil

import (
	"net"
	"os"

	"github.com/jlewi/mlkube.io/pkg/spec"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	log "github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // for gcp auth
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO(jlewi): I think this function is used to add an owner to a resource. I think we we should use this
// method to ensure all resources created for the TfJob are owned by the TfJob.
func addOwnerRefToObject(o metav1.Object, r metav1.OwnerReference) {
	o.SetOwnerReferences(append(o.GetOwnerReferences(), r))
}

func MustNewKubeClient() kubernetes.Interface {
	cfg, err := InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	return kubernetes.NewForConfigOrDie(cfg)
}

func MustNewApiExtensionsClient() apiextensionsclient.Interface {
	cfg, err := InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	return apiextensionsclient.NewForConfigOrDie(cfg)
}

// TODO(jlewi): We should rename InClusterConfig to reflect the fact that we can obtain the config from the Kube
// configuration used by kubeconfig.
func InClusterConfig() (*rest.Config, error) {
	if len(os.Getenv("USE_KUBE_CONFIG")) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		return clientcmd.BuildConfigFromFlags("", os.Getenv("USE_KUBE_CONFIG"))
	}

	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0])
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	}
	return rest.InClusterConfig()
}

func IsKubernetesResourceAlreadyExistError(err error) bool {
	return apierrors.IsAlreadyExists(err)
}

func IsKubernetesResourceNotFoundError(err error) bool {
	return apierrors.IsNotFound(err)
}

// We are using internal api types for cluster related.
func JobListOpt(clusterName string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(LabelsForJob(clusterName)).String(),
	}
}

func LabelsForJob(jobName string) map[string]string {
	return map[string]string{
		// TODO(jlewi): Need to set appropriate labels for TF.
		"tf_job": jobName,
		"app":    spec.AppLabel,
	}
}

// TODO(jlewi): CascadeDeletOptions are part of garbage collection policy.
// Do we want to use this? See
// https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
func CascadeDeleteOptions(gracePeriodSeconds int64) *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		GracePeriodSeconds: func(t int64) *int64 { return &t }(gracePeriodSeconds),
		PropagationPolicy: func() *metav1.DeletionPropagation {
			foreground := metav1.DeletePropagationForeground
			return &foreground
		}(),
	}
}

// mergeLabels merges l2 into l1. Conflicting labels will be skipped.
func mergeLabels(l1, l2 map[string]string) {
	for k, v := range l2 {
		if _, ok := l1[k]; ok {
			continue
		}
		l1[k] = v
	}
}
