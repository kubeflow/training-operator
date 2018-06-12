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

	"k8s.io/api/core/v1"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

func TestFailed(t *testing.T) {
	tfJob := newTFJob(3, 0)
	initializeTFReplicaStatuses(tfJob, tfv1alpha2.TFReplicaTypeWorker)
	pod := newBasePod("pod", tfJob, t)
	pod.Status.Phase = v1.PodFailed
	updateTFJobReplicaStatuses(tfJob, tfv1alpha2.TFReplicaTypeWorker, pod)
	if tfJob.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker].Failed != 1 {
		t.Errorf("Failed to set the failed to 1")
	}
	err := updateStatus(tfJob, tfv1alpha2.TFReplicaTypeWorker, 3)
	if err != nil {
		t.Errorf("Expected error %v to be nil", err)
	}
	found := false
	for _, condition := range tfJob.Status.Conditions {
		if condition.Type == tfv1alpha2.TFJobFailed {
			found = true
		}
	}
	if !found {
		t.Errorf("Failed condition is not found")
	}
}
