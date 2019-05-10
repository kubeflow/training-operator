// Package unstructured is the package for unstructured informer,
// which is from https://github.com/argoproj/argo/blob/master/util/unstructured/unstructured.go
// This is a temporary solution for https://github.com/kubeflow/tf-operator/issues/561
package unstructured

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	informer "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/tensorflow/v1"
	lister "github.com/kubeflow/tf-operator/pkg/client/listers/tensorflow/v1"
)

type UnstructuredInformer struct {
	informer cache.SharedIndexInformer
}

func NewTFJobInformer(resource schema.GroupVersionResource, client dynamic.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) informer.TFJobInformer {
	return &UnstructuredInformer{
		informer: newUnstructuredInformer(resource, client, namespace, resyncPeriod, indexers),
	}
}

func (f *UnstructuredInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *UnstructuredInformer) Lister() lister.TFJobLister {
	return lister.NewTFJobLister(f.Informer().GetIndexer())
}

// newUnstructuredInformer constructs a new informer for Unstructured type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func newUnstructuredInformer(resource schema.GroupVersionResource, client dynamic.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return newFilteredUnstructuredInformer(resource, client, namespace, resyncPeriod, indexers)
}

// newFilteredUnstructuredInformer constructs a new informer for Unstructured type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func newFilteredUnstructuredInformer(resource schema.GroupVersionResource, client dynamic.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return client.Resource(resource).Namespace(namespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.Resource(resource).Namespace(namespace).Watch(options)
			},
		},
		&unstructured.Unstructured{},
		resyncPeriod,
		indexers,
	)
}
