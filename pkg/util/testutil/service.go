package testutil

import (
	"fmt"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/generator"
)

func NewBaseService(name string, tfJob *tfv1alpha2.TFJob, t *testing.T) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          generator.GenLabels(GetKey(tfJob, t)),
			Namespace:       tfJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(tfJob, controllerKind)},
		},
	}
}

func NewService(tfJob *tfv1alpha2.TFJob, typ string, index int, t *testing.T) *v1.Service {
	service := NewBaseService(fmt.Sprintf("%s-%d", typ, index), tfJob, t)
	service.Labels[tfReplicaTypeLabel] = typ
	service.Labels[tfReplicaIndexLabel] = fmt.Sprintf("%d", index)
	return service
}

// NewServiceList creates count pods with the given phase for the given tfJob
func NewServiceList(count int32, tfJob *tfv1alpha2.TFJob, typ string, t *testing.T) []*v1.Service {
	services := []*v1.Service{}
	for i := int32(0); i < count; i++ {
		newService := NewService(tfJob, typ, int(i), t)
		services = append(services, newService)
	}
	return services
}

func SetServices(serviceIndexer cache.Indexer, tfJob *tfv1alpha2.TFJob, typ string, activeWorkerServices int32, t *testing.T) {
	for _, service := range NewServiceList(activeWorkerServices, tfJob, typ, t) {
		serviceIndexer.Add(service)
	}
}
