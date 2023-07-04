// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"strconv"
	"strings"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/core"
	commonutil "github.com/kubeflow/training-operator/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	succeededServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reconciler_succeeded_service_creation_total",
		Help: "The total number of succeeded service creation",
	})
	failedServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reconciler_failed_service_creation_total",
		Help: "The total number of failed service creation",
	})
)

// ServiceReconciler defines a Service Reconciler for generic training job
type ServiceReconciler struct {
	client.Client
	ReconcilerUtilInterface
	PodInterface
	JobInterface
}

// BareServiceReconciler returns a pointer of ServiceReconciler with minimal implementation
func BareServiceReconciler(client client.Client) *ServiceReconciler {
	return &ServiceReconciler{
		Client: client,
	}
}

// OverrideForServiceInterface resets ReconcilerUtilInterface, PodInterface, JobInterface for ServiceReconciler
func (r *ServiceReconciler) OverrideForServiceInterface(ui ReconcilerUtilInterface, pi PodInterface, ji JobInterface) {
	if ui != nil {
		r.ReconcilerUtilInterface = ui
	}
	if pi != nil {
		r.PodInterface = pi
	}
	if ji != nil {
		r.JobInterface = ji
	}
}

// GetPortsFromJob gets the ports of job container. Port could be nil, if distributed communication strategy doesn't need and no other ports that need to be exposed.
func (r *ServiceReconciler) GetPortsFromJob(spec *kubeflowv1.ReplicaSpec) (map[string]int32, error) {
	defaultContainerName := r.GetDefaultContainerName()
	return core.GetPortsFromJob(spec, defaultContainerName)
}

// GetServicesForJob returns all services associated with this job
func (r *ServiceReconciler) GetServicesForJob(ctx context.Context, job client.Object) ([]*corev1.Service, error) {
	svcList := &corev1.ServiceList{}
	err := r.List(ctx, svcList, client.MatchingLabels(r.GenLabels(job.GetName())))
	if err != nil {
		return nil, err
	}

	var svcs []*corev1.Service
	for idx := range svcList.Items {
		svcs = append(svcs, &svcList.Items[idx])
	}

	return svcs, nil
}

// FilterServicesForReplicaType returns service belong to a replicaType.
func (r *ServiceReconciler) FilterServicesForReplicaType(services []*corev1.Service,
	replicaType string) ([]*corev1.Service, error) {
	return core.FilterServicesForReplicaType(services, replicaType)
}

// GetServiceSlices returns the serviceSlice based on all Services listed for this job
func (r *ServiceReconciler) GetServiceSlices(services []*corev1.Service, replicas int, logger *log.Entry) [][]*corev1.Service {
	return core.GetServiceSlices(services, replicas, logger)
}

// ReconcileServices reconciles the Services for this job
func (r *ServiceReconciler) ReconcileServices(
	job client.Object,
	services []*corev1.Service,
	rtype kubeflowv1.ReplicaType,
	spec *kubeflowv1.ReplicaSpec) error {

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services, err := r.FilterServicesForReplicaType(services, rt)
	if err != nil {
		return err
	}

	// GetServiceSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have services with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a svc with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], svc with replica-index 1 and 2 are out of range and will be deleted.
	serviceSlices := r.GetServiceSlices(services, replicas, commonutil.LoggerForReplica(job, rt))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			commonutil.LoggerForReplica(job, rt).Warningf("We have too many services for %s %d", rtype, index)
		} else if len(serviceSlice) == 0 {
			commonutil.LoggerForReplica(job, rt).Infof("need to create new service: %s-%d", rtype, index)
			err = r.CreateNewService(job, rtype, spec, strconv.Itoa(index))
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current svc.
			svc := serviceSlice[0]

			// check if the index is in the valid range, if not, we should kill the svc
			if index < 0 || index >= replicas {
				err = r.DeleteService(svc.Namespace, svc.Name, job)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil

}

// CreateNewService generates Service based the job, replica info. and index and submits it to APIServer
func (r *ServiceReconciler) CreateNewService(job client.Object, rtype kubeflowv1.ReplicaType,
	spec *kubeflowv1.ReplicaSpec, index string) error {

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	// Append ReplicaTypeLabel and ReplicaIndexLabel labels.
	labels := r.GenLabels(job.GetName())
	labels[kubeflowv1.ReplicaTypeLabel] = rt
	labels[kubeflowv1.ReplicaIndexLabel] = index

	ports, err := r.GetPortsFromJob(spec)
	if err != nil {
		return err
	}

	service := &corev1.Service{
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports:     []corev1.ServicePort{},
		},
	}

	// Add service ports to headless service
	for name, port := range ports {
		svcPort := corev1.ServicePort{Name: name, Port: port}
		service.Spec.Ports = append(service.Spec.Ports, svcPort)
	}

	service.Name = core.GenGeneralName(job.GetName(), rt, index)
	service.Namespace = job.GetNamespace()
	service.Labels = labels
	// Create OwnerReference.
	err = controllerutil.SetControllerReference(job, service, r.GetScheme())
	if err != nil {
		return err
	}

	r.DecorateService(rt, service, job)

	err = r.Create(context.Background(), service)
	if err != nil && errors.IsTimeout(err) {
		succeededServiceCreationCount.Inc()
		return nil
	} else if err != nil {
		failedServiceCreationCount.Inc()
		return err
	}
	succeededServiceCreationCount.Inc()
	return nil
}

// DeleteService deletes a Service specified by its name and namespace from APIServer
func (r *ServiceReconciler) DeleteService(ns string, name string, job client.Object) error {
	svc := &corev1.Service{}
	svc.Name = name
	svc.Namespace = ns
	err := r.Delete(context.Background(), svc)
	if err == nil {
		deletedPodsCount.Inc()
	}
	return err
}

// DecorateService decorates the Service before it's submitted to APIServer
func (r *ServiceReconciler) DecorateService(rtype string, svc *corev1.Service, job client.Object) {
	// Default implementation applies nothing to podTemplate
	// return
}
