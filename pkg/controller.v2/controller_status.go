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
	"strconv"

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
	// tfJobRestartingReason is added in a tfjob when it is restarting.
	tfJobRestartingReason = "TFJobRestarting"
	//workerCreatedReason is added in a status when it is  created.
	workerCreatedReason = "WorkerCreated"
	//workerSucceededReason is added in a status when it is  succeeded.
	workerSucceededReason = "WorkerSucceeded"
	//workerSucceededReason is added in a status when it is  running.
	workerRunningReason = "WorkerRunning"
	//workerFailedReason is added in a status when it is failed.
	workerFailedReason = "WorkerFailed"
	//workerPendingReason is added in a status when it is  pending.
	workerPendingReason = "WorkerPending"
	//psCreatedReason is added in a status when it is  created.
	psCreatedreason = "PSCreated"
	//psSucceededReason is added in a status when it is  succeeded.
	psSucceededReason = "PSSucceeded"
	//psRunningReason is added in a status when it is  running.
	psRunningReason = "PSRunning"
	//psFailedReason is added in a status when it is  failed.
	psFailedReason = "PSFailed"
	//psPendingReason is added in a status when it is  pending.
	psPendingReason = "PSPending"
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

func (tc *TFJobController) updateStatusNew(tfjob *tfv1alpha2.TFJob, rstatus map[string]v1.PodPhase, wreplicas int, preplicas int) error {
	if preplicas == 0 {
		tc.updateStatus(tfjob, tfv1alpha2.TFReplicaTypeWorker, wreplicas)
	} else {
		status := make(map[string]int)

		initReplicasStatusesNum(status)
		workerKey := string(tfv1alpha2.TFReplicaTypeWorker) + "-0"
		switch rstatus[workerKey] {
		case v1.PodRunning:
			status[workerRunningReason] = 1
		case v1.PodSucceeded:
			status[workerSucceededReason] = 1
		case v1.PodPending:
			status[workerPendingReason] = 1
		case v1.PodFailed:
			status[workerFailedReason] = 1
		}
		for i := 0; i < preplicas; i++ {
			psKey := string(tfv1alpha2.TFReplicaTypePS) + "-" + strconv.Itoa(i)
			switch rstatus[psKey] {
			case v1.PodRunning:
				status[psRunningReason]++
			case v1.PodFailed:
				status[psFailedReason]++
			case v1.PodPending:
				status[psPendingReason]++
			}
		}
		if status[psRunningReason] == preplicas && status[workerRunningReason] == 1 {
			//Running
			msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
			now := metav1.Now()
			tfjob.Status.CompletionTime = &now
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		}
		if status[psRunningReason] == preplicas && status[workerSucceededReason] == 1 {
			//Succeeded
			msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
			now := metav1.Now()
			tfjob.Status.CompletionTime = &now
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}

		}
		if status[psFailedReason] != 0 || status[workerFailedReason] != 0 {
			//Failed
			msg := fmt.Sprintf("TFJob %s is failed.", tfjob.Name)
			now := metav1.Now()
			tfjob.Status.CompletionTime = &now
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobFailed, tfJobFailedReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}

		}
		if status[psRunningReason] == preplicas && status[workerPendingReason] == 1 && tfjob.Status.Conditions[len(tfjob.Status.Conditions)-1].Type == tfv1alpha2.TFJobFailed {
			//Restarting
			msg := fmt.Sprintf("TFJob %s is restarting ", tfjob.Name)
			now := metav1.Now()
			tfjob.Status.CompletionTime = &now
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobRestarting, tfJobRestartingReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}

		}
	}
	return nil
}

// initReplicasStatusesNum initializes the PS or Worker for status.
func initReplicasStatusesNum(status map[string]int) {
	status[workerPendingReason] = 0
	status[workerRunningReason] = 0
	status[workerFailedReason] = 0
	status[workerSucceededReason] = 0
	status[workerCreatedReason] = 0
	status[psCreatedreason] = 0
	status[psPendingReason] = 0
	status[psRunningReason] = 0
	status[psFailedReason] = 0
	status[psSucceededReason] = 0
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

func addTFJobReplicaStatuses(rtype tfv1alpha2.TFReplicaType, index int, pod *v1.Pod, rstatus map[string]v1.PodPhase) {
	key := string(rtype) + "-" + strconv.Itoa(index)
	for _, val := range pod.Status.ContainerStatuses {
		if val.State.Waiting != nil {
			rstatus[key] = v1.PodFailed
			return
		}
	}
	rstatus[key] = pod.Status.Phase
}

// increaseTFJobReplicaStatusesActive increases active in TFJobReplicaStatuses.
func increaseTFJobReplicaStatusesActive(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType) {
	tfjob.Status.TFReplicaStatuses[rtype].Active++
}
