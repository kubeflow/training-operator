package pytorch

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/apis/pytorch/validation"
	jobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	jobinformersv1alpha2 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/pytorch/v1alpha2"
	lister "github.com/kubeflow/tf-operator/pkg/client/listers/pytorch/v1alpha2"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
	"github.com/kubeflow/tf-operator/pkg/util/unstructured"
)

const (
	resyncPeriod     = 30 * time.Second
	failedMarshalMsg = "Failed to marshal the object to PyTorchJob: %v"
)

var (
	errGetFromKey    = fmt.Errorf("Failed to get PyTorchJob from key")
	errNotExists     = fmt.Errorf("The object is not found")
	errFailedMarshal = fmt.Errorf("Failed to marshal the object to PyTorchJob")
)

type UnstructuredPyTorchInformer struct {
	informer cache.SharedIndexInformer
}

func (f *UnstructuredPyTorchInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *UnstructuredPyTorchInformer) Lister() lister.PyTorchJobLister {
	return lister.NewPyTorchJobLister(f.Informer().GetIndexer())
}

func NewUnstructuredPyTorchJobInformer(restConfig *restclientset.Config) jobinformersv1alpha2.PyTorchJobInformer {
	dynClientPool := dynamic.NewDynamicClientPool(restConfig)
	dclient, err := dynClientPool.ClientForGroupVersionKind(v1alpha2.SchemeGroupVersionKind)
	if err != nil {
		panic(err)
	}
	resource := &metav1.APIResource{
		Name:         v1alpha2.Plural,
		SingularName: v1alpha2.Singular,
		Namespaced:   true,
		Group:        v1alpha2.GroupName,
		Version:      v1alpha2.GroupVersion,
	}
	informer := &UnstructuredPyTorchInformer{
		informer: unstructured.NewUnstructuredInformer(
			resource,
			dclient,
			metav1.NamespaceAll,
			resyncPeriod,
			cache.Indexers{},
		),
	}
	return informer
}

// NewPyTorchJobInformer returns PyTorchJobInformer from the given factory.
func (pc *PyTorchController) NewPyTorchJobInformer(jobInformerFactory jobinformers.SharedInformerFactory) jobinformersv1alpha2.PyTorchJobInformer {
	return jobInformerFactory.Pytorch().V1alpha2().PyTorchJobs()
}

func (pc *PyTorchController) getPyTorchJobFromName(namespace, name string) (*v1alpha2.PyTorchJob, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	return pc.getPyTorchJobFromKey(key)
}

func (pc *PyTorchController) getPyTorchJobFromKey(key string) (*v1alpha2.PyTorchJob, error) {
	// Check if the key exists.
	obj, exists, err := pc.jobInformer.GetIndexer().GetByKey(key)
	logger := pylogger.LoggerForKey(key)
	if err != nil {
		logger.Errorf("Failed to get PyTorchJob '%s' from informer index: %+v", key, err)
		return nil, errGetFromKey
	}
	if !exists {
		// This happens after a job was deleted, but the work queue still had an entry for it.
		return nil, errNotExists
	}

	job, err := jobFromUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func jobFromUnstructured(obj interface{}) (*v1alpha2.PyTorchJob, error) {
	// Check if the spec is valid.
	un, ok := obj.(*metav1unstructured.Unstructured)
	if !ok {
		log.Errorf("The object in index is not an unstructured; %+v", obj)
		return nil, errGetFromKey
	}
	var job v1alpha2.PyTorchJob
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &job)
	logger := pylogger.LoggerForUnstructured(un, v1alpha2.Kind)
	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}

	err = validation.ValidateAlphaTwoPyTorchJobSpec(&job.Spec)
	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}
	return &job, nil
}

func unstructuredFromPyTorchJob(obj interface{}, job *v1alpha2.PyTorchJob) error {
	un, ok := obj.(*metav1unstructured.Unstructured)
	logger := pylogger.LoggerForJob(job)
	if !ok {
		logger.Warn("The object in index isn't type Unstructured")
		return errGetFromKey
	}

	var err error
	un.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		logger.Error("The PyTorchJob convert failed")
		return err
	}
	return nil

}
