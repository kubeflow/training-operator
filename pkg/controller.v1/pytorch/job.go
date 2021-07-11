package controllers

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonutil "github.com/kubeflow/common/pkg/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	pytorchv1 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1"
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
)

var defaultCleanPodPolicy = commonv1.CleanPodPolicyNone

// UpdateJobStatusInApiServer updates the job status in to cluster.
func (r *PyTorchJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	pytorchjob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", job)
	}

	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !reflect.DeepEqual(&pytorchjob.Status, jobStatus) {
		pytorchjob = pytorchjob.DeepCopy()
		pytorchjob.Status = *jobStatus.DeepCopy()
	}

	result := r.Update(context.Background(), pytorchjob)

	if result != nil {
		r.Log.WithValues(pytorchv1.Singular, types.NamespacedName{
			Namespace: pytorchjob.GetNamespace(),
			Name:      pytorchjob.GetName(),
		})
		return result
	}

	return nil
}

// UpdateJobStatus updates the job status and job conditions
func (r *PyTorchJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	pytorchjob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", job)
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logrus.Infof("XGBoostJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			pytorchjob.Name, rtype, expected, running, succeeded, failed)

		if rtype == commonv1.ReplicaType(pytorchv1.PyTorchReplicaTypeMaster) {
			if running > 0 {
				msg := fmt.Sprintf("XGBoostJob %s is running.", pytorchjob.Name)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
			}
			// when master is succeed, the job is finished.
			if expected == 0 {
				msg := fmt.Sprintf("XGBoostJob %s is successfully completed.", pytorchjob.Name)
				logrus.Info(msg)
				r.Recorder.Event(pytorchjob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, commonutil.JobSucceededReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
				return nil
			}
		}
		if failed > 0 {
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				msg := fmt.Sprintf("XGBoostJob %s is restarting because %d %s replica(s) failed.", pytorchjob.Name, failed, rtype)
				r.Recorder.Event(pytorchjob, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
			} else {
				msg := fmt.Sprintf("XGBoostJob %s is failed because %d %s replica(s) failed.", pytorchjob.Name, failed, rtype)
				r.Recorder.Event(pytorchjob, corev1.EventTypeNormal, commonutil.JobFailedReason, msg)
				if pytorchjob.Status.CompletionTime == nil {
					now := metav1.Now()
					pytorchjob.Status.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, commonutil.JobFailedReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
			}
		}
	}

	// Some workers are still running, leave a running condition.
	msg := fmt.Sprintf("PyTorchJob %s is running.", pytorchjob.Name)
	commonutil.LoggerForJob(pytorchjob).Infof(msg)

	if err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg); err != nil {
		commonutil.LoggerForJob(pytorchjob).Error(err, "failed to update XGBoost Job conditions")
		return err
	}

	return nil
}

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		pytorchjob, ok := e.Object.(*pytorchv1.PyTorchJob)
		if !ok {
			return true
		}
		scheme.Scheme.Default(pytorchjob)
		msg := fmt.Sprintf("xgboostJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		//specific the run policy

		if pytorchjob.Spec.CleanPodPolicy == nil {
			pytorchjob.Spec.CleanPodPolicy = new(commonv1.CleanPodPolicy)
			pytorchjob.Spec.CleanPodPolicy = &defaultCleanPodPolicy
		}

		if err := commonutil.UpdateJobConditions(&pytorchjob.Status, commonv1.JobCreated, "PyTorchJobCreated", msg); err != nil {
			logrus.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
