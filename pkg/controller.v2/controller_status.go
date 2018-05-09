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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

const (
	// tfJobCreatedReason is added in a tfjob when it is created.
	tfJobCreatedReason = "TFJobCreated"
	// tfJobSucceededReason is added in a tfjob when it is succeeded.
	tfJobSucceededReason = "TFJobSucceeded"
	// tfJobSucceededReason is added in a tfjob when it is running.
	tfJobRunningReason = "TFJobRunning"
	// tfJobSucceededReason is added in a tfjob when it is failed.
	tfJobFailedReason = "TFJobFailed"
)

// updateStatus updates the status of the tfjob.
func (tc *TFJobController) updateStatus(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType, replicas int) error {
	// Expect to have `replicas - succeeded` pods alive.
	expected := replicas - int(tfjob.Status.TFReplicaStatuses[rtype].Succeeded)
	running := int(tfjob.Status.TFReplicaStatuses[rtype].Active)
	failed := int(tfjob.Status.TFReplicaStatuses[rtype].Failed)

	if rtype == tfv1alpha2.TFReplicaTypeWorker {
		// All workers are running, set StartTime.
		if running == replicas {
			now := metav1.Now()
			tfjob.Status.StartTime = &now
		}

		// Some workers are still running, leave a running condition.
		if running > 0 {
			msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		}

		// All workers are succeeded, leave a succeeded condition.
		if expected == 0 {
			msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
			now := metav1.Now()
			tfjob.Status.CompletionTime = &now
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		}
	}

	// Some workers or pss are failed , leave a failed condition.
	if failed > 0 {
		msg := fmt.Sprintf("TFJob %s is failed.", tfjob.Name)
		err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobFailed, tfJobFailedReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}
	}
	return nil
}

// initializeTFReplicaStatuses initializes the TFReplicaStatuses for replica.
func initializeTFReplicaStatuses(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType) {
	if tfjob.Status.TFReplicaStatuses == nil {
		tfjob.Status.TFReplicaStatuses = make(map[tfv1alpha2.TFReplicaType]*tfv1alpha2.TFReplicaStatus)
	}

	tfjob.Status.TFReplicaStatuses[rtype] = &tfv1alpha2.TFReplicaStatus{}
}

// updateTFJobReplicaStatuses updates the TFJobReplicaStatuses according to the pod.
func updateTFJobReplicaStatuses(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType, pod *v1.Pod) {
	switch pod.Status.Phase {
	case v1.PodRunning:
		tfjob.Status.TFReplicaStatuses[rtype].Active++
	case v1.PodSucceeded:
		tfjob.Status.TFReplicaStatuses[rtype].Succeeded++
	case v1.PodFailed:
		tfjob.Status.TFReplicaStatuses[rtype].Failed++
	}
}

// increaseTFJobReplicaStatusesActive increases active in TFJobReplicaStatuses.
func increaseTFJobReplicaStatusesActive(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType) {
	tfjob.Status.TFReplicaStatuses[rtype].Active++
}
