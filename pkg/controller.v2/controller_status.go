package controller

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

func (tc *TFJobController) UpdateStatus(tfjob *tfv1alpha2.TFJob, replicas int, succeeded, failed int32) {
	// Expect to have `replicas - succeeded` pods alive.
	expected := replicas - succeeded

	// All workers are succeeded, leave a succeeded condition.
	if expected == 0 && rtype == tfv1alpha2.TFReplicaTypeWorker {
		msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
		err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}
	}

	// Some workers are still running, leave a running condition.
	if running > 0 && rtype == tfv1alpha2.TFReplicaTypeWorker {
		msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
		err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}
	}

	// All workers are running, set StartTime
	if running == replicas && rtype == tfv1alpha2.TFReplicaTypeWorker {
		now := metav1.Now()
		tfjob.Status.StartTime = &now
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

	if tfjob.Status.TFReplicaStatuses == nil {
		tfjob.Status.TFReplicaStatuses = make(map[tfv1alpha2.TFReplicaType]*tfv1alpha2.TFReplicaStatus)
	}

	if _, ok := tfjob.Status.TFReplicaStatuses[rtype]; !ok {
		tfjob.Status.TFReplicaStatuses[rtype] = &tfv1alpha2.TFReplicaStatus{}
	}

	// Update the active status since we have created -diff pods during the loop.
	tfjob.Status.TFReplicaStatuses[rtype].Active = expected
	tfjob.Status.TFReplicaStatuses[rtype].Succeeded = succeeded
	tfjob.Status.TFReplicaStatuses[rtype].Failed = failed
}
