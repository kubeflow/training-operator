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
package controller

import (
	"fmt"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
)

func newBaseService(name string, tfJob *tfv1alpha2.TFJob, t *testing.T) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          genLabels(getKey(tfJob, t)),
			Namespace:       tfJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(tfJob, controllerKind)},
		},
	}
}

func newService(tfJob *tfv1alpha2.TFJob, typ string, index int, t *testing.T) *v1.Service {
	service := newBaseService(fmt.Sprintf("%s-%d", typ, index), tfJob, t)
	service.Labels[tfReplicaTypeLabel] = typ
	service.Labels[tfReplicaIndexLabel] = fmt.Sprintf("%d", index)
	return service
}

// create count pods with the given phase for the given tfJob
func newServiceList(count int32, tfJob *tfv1alpha2.TFJob, typ string, t *testing.T) []*v1.Service {
	services := []*v1.Service{}
	for i := int32(0); i < count; i++ {
		newService := newService(tfJob, typ, int(i), t)
		services = append(services, newService)
	}
	return services
}

func setServices(serviceIndexer cache.Indexer, tfJob *tfv1alpha2.TFJob, typ string, activeWorkerServices int32, t *testing.T) {
	for _, service := range newServiceList(activeWorkerServices, tfJob, typ, t) {
		serviceIndexer.Add(service)
	}
}

func TestAddService(t *testing.T) {
	// Prepare the clientset and controller for the test.
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)
	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &tfv1alpha2.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
	ctr.tfJobInformerSynced = alwaysReady
	ctr.podInformerSynced = alwaysReady
	ctr.serviceInformerSynced = alwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(threadCount, stopCh)
	}
	go run(stopCh)

	var key string
	syncChan := make(chan string)
	ctr.syncHandler = func(tfJobKey string) (bool, error) {
		key = tfJobKey
		<-syncChan
		return true, nil
	}

	tfJob := newTFJob(1, 0)
	unstructured, err := convertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}

	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}
	service := newService(tfJob, labelWorker, 0, t)
	ctr.addService(service)

	syncChan <- "sync"
	if key != getKey(tfJob, t) {
		t.Errorf("Failed to enqueue the TFJob %s: expected %s, got %s", tfJob.Name, getKey(tfJob, t), key)
	}
	close(stopCh)
}
