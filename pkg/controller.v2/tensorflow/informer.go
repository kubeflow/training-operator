package tensorflow

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

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/apis/tensorflow/validation"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	tfjobinformersv1alpha2 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/kubeflow/v1alpha2"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
	"github.com/kubeflow/tf-operator/pkg/util/unstructured"
)

const (
	resyncPeriod     = 30 * time.Second
	failedMarshalMsg = "Failed to marshal the object to TFJob: %v"
)

var (
	errGetFromKey    = fmt.Errorf("failed to get TFJob from key")
	errNotExists     = fmt.Errorf("the object is not found")
	errFailedMarshal = fmt.Errorf("failed to marshal the object to TFJob")
)

func NewUnstructuredTFJobInformer(restConfig *restclientset.Config, namespace string) tfjobinformersv1alpha2.TFJobInformer {
	dynClientPool := dynamic.NewDynamicClientPool(restConfig)
	dclient, err := dynClientPool.ClientForGroupVersionKind(tfv1alpha2.SchemeGroupVersionKind)
	if err != nil {
		panic(err)
	}
	resource := &metav1.APIResource{
		Name:         tfv1alpha2.Plural,
		SingularName: tfv1alpha2.Singular,
		Namespaced:   true,
		Group:        tfv1alpha2.GroupName,
		Version:      tfv1alpha2.GroupVersion,
	}
	informer := unstructured.NewTFJobInformer(
		resource,
		dclient,
		namespace,
		resyncPeriod,
		cache.Indexers{},
	)
	return informer
}

// NewTFJobInformer returns TFJobInformer from the given factory.
func (tc *TFController) NewTFJobInformer(tfJobInformerFactory tfjobinformers.SharedInformerFactory) tfjobinformersv1alpha2.TFJobInformer {
	return tfJobInformerFactory.Kubeflow().V1alpha2().TFJobs()
}

func (tc *TFController) getTFJobFromName(namespace, name string) (*tfv1alpha2.TFJob, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	return tc.getTFJobFromKey(key)
}

func (tc *TFController) getTFJobFromKey(key string) (*tfv1alpha2.TFJob, error) {
	logger := tflogger.LoggerForKey(key)
	// Check if the key exists.
	obj, exists, err := tc.tfJobInformer.GetIndexer().GetByKey(key)
	if err != nil {
		logger.Errorf("Failed to get TFJob '%s' from informer index: %+v", key, err)
		return nil, errGetFromKey
	}
	if !exists {
		// This happens after a tfjob was deleted, but the work queue still had an entry for it.
		return nil, errNotExists
	}

	return tfJobFromUnstructured(obj)
}

func tfJobFromUnstructured(obj interface{}) (*tfv1alpha2.TFJob, error) {
	// Check if the spec is valid.
	un, ok := obj.(*metav1unstructured.Unstructured)
	if !ok {
		log.Errorf("The object in index is not an unstructured; %+v", obj)
		return nil, errGetFromKey
	}
	var tfjob tfv1alpha2.TFJob
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &tfjob)
	logger := tflogger.LoggerForUnstructured(un, tfv1alpha2.Kind)
	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}
	// This is a simple validation for TFJob to close
	// https://github.com/kubeflow/tf-operator/issues/641
	// TODO(gaocegege): Add more validation here.
	err = validation.ValidateAlphaTwoTFJobSpec(&tfjob.Spec)
	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}
	return &tfjob, nil
}

func unstructuredFromTFJob(obj interface{}, tfJob *tfv1alpha2.TFJob) error {
	un, ok := obj.(*metav1unstructured.Unstructured)
	logger := tflogger.LoggerForJob(tfJob)
	if !ok {
		logger.Warn("The object in index isn't type Unstructured")
		return errGetFromKey
	}

	var err error
	un.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(tfJob)
	if err != nil {
		logger.Error("The TFJob convert failed")
		return err
	}
	return nil

}
