// Copyright 2018 The Kubeflow Authors
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

// Package controller provides a Kubernetes controller for a TFJob resource.
package tensorflow

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	common "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta1"
	tfv1beta1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta1"
	"github.com/kubeflow/tf-operator/pkg/common/jobcontroller"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
)

// reconcileServices checks and updates services for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting services.
func (tc *TFController) reconcileServices(
	tfjob *tfv1beta1.TFJob,
	services []*v1.Service,
	rtype tfv1beta1.TFReplicaType,
	spec *common.ReplicaSpec) error {

	// Convert TFReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services, err := tc.FilterServicesForReplicaType(services, rt)
	if err != nil {
		return err
	}

	serviceSlices := tc.GetServiceSlices(services, replicas, tflogger.LoggerForReplica(tfjob, rt))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			tflogger.LoggerForReplica(tfjob, rt).Warningf("We have too many services for %s %d", rt, index)
			// TODO(gaocegege): Kill some services.
		} else if len(serviceSlice) == 0 {
			tflogger.LoggerForReplica(tfjob, rt).Infof("need to create new service: %s-%d", rt, index)
			err = tc.createNewService(tfjob, rtype, strconv.Itoa(index), spec)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// createNewService creates a new service for the given index and type.
func (tc *TFController) createNewService(tfjob *tfv1beta1.TFJob, rtype tfv1beta1.TFReplicaType, index string, spec *common.ReplicaSpec) error {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}

	// Convert TFReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	expectationServicesKey := jobcontroller.GenExpectationServicesKey(tfjobKey, rt)
	err = tc.Expectations.ExpectCreations(expectationServicesKey, 1)
	if err != nil {
		return err
	}

	// Create OwnerReference.
	controllerRef := tc.GenOwnerReference(tfjob)

	// Append tfReplicaTypeLabel and tfReplicaIndexLabel labels.
	labels := tc.GenLabels(tfjob.Name)
	labels[tfReplicaTypeLabel] = rt
	labels[tfReplicaIndexLabel] = index

	port, err := GetPortFromTFJob(tfjob, rtype)
	if err != nil {
		return err
	}

	service := &v1.Service{
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []v1.ServicePort{
				{
					Name: tfv1beta1.DefaultPortName,
					Port: port,
				},
			},
		},
	}

	service.Name = jobcontroller.GenGeneralName(tfjob.Name, rt, index)
	service.Labels = labels

	err = tc.ServiceControl.CreateServicesWithControllerRef(tfjob.Namespace, service, tfjob, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Service is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the service keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// service when the expectation expires.
		return nil
	} else if err != nil {
		return err
	}
	return nil
}
