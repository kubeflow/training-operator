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
	"testing"
	"time"

	"k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
)

func TestAddPod(t *testing.T) {
	// Prepare the clientset and controller for the test.
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &tfv1alpha2.SchemeGroupVersion,
		},
	},
	)
	controller, kubeInformerFactory, tfJobInformerFactory := newTFJobControllerFromClient(kubeClientSet, tfJobClientSet, NoResyncPeriodFunc)
	controller.tfJobListerSynced = alwaysReady
	controller.podListerSynced = alwaysReady
	controller.serviceListerSynced = alwaysReady
	podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	tfJobIndexer := tfJobInformerFactory.Kubeflow().V1alpha2().TFJobs().Informer().GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		controller.Run(threadCount, stopCh)
	}
	go run(stopCh)

	var key string
	controller.syncHandler = func(tfJobKey string) (bool, error) {
		key = tfJobKey
		return true, nil
	}

	tfJob := newTFJob(1, 0)
	tfJobIndexer.Add(tfJob)
	pod := newPod(tfJob, labelWorker, 0, t)
	podIndexer.Add(pod)

	controller.addPod(pod)
	time.Sleep(sleepInterval)
	if key != getKey(tfJob, t) {
		t.Errorf("Failed to enqueue the TFJob %s: expected %s, got %s", tfJob.Name, getKey(tfJob, t), key)
	}
	close(stopCh)
}
