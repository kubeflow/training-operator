package mxnet

import (
	"context"
	"fmt"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	mxv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"reflect"
	"time"
)

const (
	// mxJobCreatedReason is added in a mxjob when it is created.
	mxJobCreatedReason = "MXJobCreated"
	// mxJobSucceededReason is added in a mxjob when it is succeeded.
	mxJobSucceededReason = "MXJobSucceeded"
	// mxJobRunningReason is added in a mxjob when it is running.
	mxJobRunningReason = "MXJobRunning"
	// mxJobFailedReason is added in a mxjob when it is failed.
	mxJobFailedReason = "MXJobFailed"
	// mxJobRestarting is added in a mxjob when it is restarting.
	mxJobRestartingReason = "MXJobRestarting"
)

func (r *MXJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	mxjob, ok := job.(*mxv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxjob)
	}

	mxjobKey, err := KeyFunc(mxjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for mxjob object %#v: %v", mxjob, err))
		return err
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]

		// Expect to have `replicas - succeeded` pods alive.
		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		r.Log.Info(fmt.Sprintf("MXJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			mxjob.Name, rtype, expected, running, succeeded, failed))

		if mxjob.Status.StartTime == nil {
			now := metav1.Now()
			mxjob.Status.StartTime = &now
			// enqueue a sync to check if job past ActiveDeadlineSeconds
			if mxjob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
				logrus.Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *mxjob.Spec.RunPolicy.ActiveDeadlineSeconds)
				r.WorkQueue.AddAfter(mxjobKey, time.Duration(*mxjob.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
			}
		}

		if running > 0 {
			msg := fmt.Sprintf("MXJob %s is running.", mxjob.Name)
			err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, mxJobRunningReason, msg)
			if err != nil {
				logrus.Infof("Append mxjob condition error: %v", err)
				return err
			}
		}
		if expected == 0 {
			msg := fmt.Sprintf("MXJob %s is successfully completed.", mxjob.Name)
			r.Recorder.Event(mxjob, corev1.EventTypeNormal, mxJobSucceededReason, msg)
			if mxjob.Status.CompletionTime == nil {
				now := metav1.Now()
				mxjob.Status.CompletionTime = &now
			}
			err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, mxJobSucceededReason, msg)
			if err != nil {
				logrus.Infof("Append mxjob condition error: %v", err)
				return err
			}
		}

		if failed > 0 {
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				msg := fmt.Sprintf("mxjob %s is restarting because %d %s replica(s) failed.", mxjob.Name, failed, rtype)
				r.Recorder.Event(mxjob, corev1.EventTypeWarning, mxJobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, mxJobRestartingReason, msg)
				if err != nil {
					logrus.Infof("Append job condition error: %v", err)
					return err
				}
			} else {
				msg := fmt.Sprintf("mxjob %s is failed because %d %s replica(s) failed.", mxjob.Name, failed, rtype)
				r.Recorder.Event(mxjob, corev1.EventTypeNormal, mxJobFailedReason, msg)
				if mxjob.Status.CompletionTime == nil {
					now := metav1.Now()
					mxjob.Status.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, mxJobFailedReason, msg)
				if err != nil {
					logrus.Infof("Append job condition error: %v", err)
					return err
				}
			}
		}
	}

	return nil
}

// UpdateJobStatusInApiServer updates the status of the given MXJob.
func (r *MXJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	mxJob, ok := job.(*mxv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	if !reflect.DeepEqual(&mxJob.Status, jobStatus) {
		mxJob = mxJob.DeepCopy()
		mxJob.Status = *jobStatus.DeepCopy()
	}

	if err := r.Update(context.Background(), mxJob); err != nil {
		logrus.Error(err, " failed to update MxJob conditions in the API server")
		return err
	}

	return nil
}
