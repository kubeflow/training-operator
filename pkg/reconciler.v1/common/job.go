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
	"fmt"
	"reflect"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	commonv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/core"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/k8sutil"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	GroupName = "kubeflow.org"

	ReasonKey        = "reason"
	ReasonJobDeleted = "job deleted"

	MsgReconcileCancelled = "Reconcile Cancelled"
	MsgReconcileStart     = "Reconcile Starts"

	MsgGetPodsFailed     = "Get Pods Failed"
	MsgGetServicesFailed = "Get Services Failed"

	MsgBackoffLimitReachedTemplate   = "Job %s has failed because it has reached the specified backoff limit"
	MsgActiveDeadlineReachedTemplate = "Job %s has failed because it was active longer than specified deadline"

	ErrUpdateJobConditionsFailed = "failed to update job conditions"

	ErrUpdateJobErrorTemplate                    = "UpdateJobStatus error %v"
	ErrAppendJobConditionTemplate                = "Append job condition error %v"
	ErrReconcilePodsTemplate                     = "ReconcilePods error %v"
	ErrReconcileServicesTemplate                 = "ReconcileServices error %v"
	ErrReconcileGangTemplate                     = "ReconcilePodGroups error %v"
	ErrGetReplicasStatusFromStatusFailedTemplate = "failed to get ReplicasStatus for %s from status"

	WarnDefaultImplementationTemplate = "Warning: executing default implementation for JobReconciler.%s"
	WarnNotCountedInBackoffLimit      = "The restart policy of replica %v of the job %v is not OnFailure or Always. Not counted in backoff limit."
)

// JobReconciler defines a Reconciler dealing with generic training job
type JobReconciler struct {
	client.Client
	ReconcilerUtilInterface
	PodInterface
	ServiceInterface
	GangSchedulingInterface
	counter *commonutil.Counter
}

// BareJobReconciler returns the pointer of a JobReconciler with minimal implementation
func BareJobReconciler(client client.Client) *JobReconciler {
	return &JobReconciler{
		Client:  client,
		counter: commonutil.NewCounter(),
	}
}

// OverrideForJobInterface resets ReconcilerUtilInterface, PodInterface, ServiceInterface, GangSchedulingInterface used in JobReconciler
func (r *JobReconciler) OverrideForJobInterface(ui ReconcilerUtilInterface, pi PodInterface, si ServiceInterface, gi GangSchedulingInterface) {
	if ui != nil {
		r.ReconcilerUtilInterface = ui
	}
	if pi != nil {
		r.PodInterface = pi
	}
	if si != nil {
		r.ServiceInterface = si
	}
	if gi != nil {
		r.GangSchedulingInterface = gi
	}
}

// GenLabels returns labels used for this job (based on the name of this generic training job)
func (r *JobReconciler) GenLabels(jobName string) map[string]string {
	jobName = strings.Replace(jobName, "/", "-", -1)
	return map[string]string{
		commonv1.OperatorNameLabel: r.GetReconcilerName(),
		commonv1.JobNameLabel:      jobName,
	}
}

// GetGroupNameLabelValue returns the Group Name for the generic training job, which is "kubeflow.org"
func (r *JobReconciler) GetGroupNameLabelValue() string {
	return GroupName
}

// ReconcileJob reconciles generic training job
func (r *JobReconciler) ReconcileJob(
	ctx context.Context,
	job client.Object,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	status *commonv1.JobStatus,
	runPolicy *commonv1.RunPolicy) error {

	logger := r.GetLogger(job)
	logger.Info(MsgReconcileStart)

	oldStatus := status.DeepCopy()

	var err error
	if r.ShouldCleanUp(*status) {
		if err = r.CleanupResources(runPolicy, *status, job); err != nil {
			return err
		}
		if err = r.CleanupJob(runPolicy, *status, job); err != nil {
			return err
		}
		if r.IsJobSucceeded(*status) {
			r.SetStatusForSuccessJob(status)
		}
		if !reflect.DeepEqual(*oldStatus, *status) {
			return r.UpdateJobStatusInAPIServer(ctx, job)
		}
		return nil
	}

	pods, err := r.GetPodsForJob(ctx, job)
	if err != nil {
		logger.Info(MsgGetPodsFailed)
		return err
	}

	services, err := r.GetServicesForJob(ctx, job)
	if err != nil {
		logger.Info(MsgGetServicesFailed)
		return err
	}

	previousRetry, _ := r.counter.Counts(types.NamespacedName{
		Namespace: job.GetNamespace(),
		Name:      job.GetName(),
	}.String())
	if previousRetry < 0 {
		// TODO: may be we should abort here?
		previousRetry = 0
	}

	activePods := k8sutil.FilterActivePods(pods)
	r.RecordAbnormalPods(activePods, job)

	active := int32(len(activePods))
	failed := k8sutil.FilterPodCount(pods, corev1.PodFailed)
	totalReplicas := k8sutil.GetTotalReplicas(replicas)
	prevReplicasFailedNum := k8sutil.GetTotalFailedReplicas(status.ReplicaStatuses)

	var failureMessage string
	jobExceedsLimit := false
	exceedsBackoffLimit := false
	pastBackoffLimit := false

	if runPolicy.BackoffLimit != nil {
		jobHasNewFailure := failed > prevReplicasFailedNum
		exceedsBackoffLimit = jobHasNewFailure && (active != totalReplicas) &&
			(int32(previousRetry)+1 > *runPolicy.BackoffLimit)

		pastBackoffLimit, err = r.PastBackoffLimit(job.GetName(), runPolicy, replicas, pods)
		if err != nil {
			return err
		}
	}

	if exceedsBackoffLimit || pastBackoffLimit {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncJob
		jobExceedsLimit = true
		failureMessage = fmt.Sprintf(MsgBackoffLimitReachedTemplate, job.GetName())
	} else if r.PastActiveDeadline(runPolicy, status) {
		failureMessage = fmt.Sprintf(MsgActiveDeadlineReachedTemplate, job.GetName())
		jobExceedsLimit = true
	}

	if jobExceedsLimit {
		if status.CompletionTime == nil {
			now := metav1.Now()
			status.CompletionTime = &now
		}
		if err = r.CleanupResources(runPolicy, *status, job); err != nil {
			return err
		}
		if err = r.CleanupJob(runPolicy, *status, job); err != nil {
			return err
		}
		if r.IsJobSucceeded(*status) {
			r.SetStatusForSuccessJob(status)
		}

		r.GetRecorder().Event(job, corev1.EventTypeNormal, commonutil.JobFailedReason, failureMessage)

		if err = commonutil.UpdateJobConditions(status, commonv1.JobFailed, commonutil.JobFailedReason, failureMessage); err != nil {
			logrus.Infof(ErrAppendJobConditionTemplate, err)
			return err
		}

		return r.UpdateJobStatusInAPIServer(ctx, job)
	}

	if r.GangSchedulingEnabled() {
		err = r.ReconcilePodGroup(ctx, job, runPolicy, replicas)
		if err != nil {
			logrus.Warnf(ErrReconcileGangTemplate, err)
			return err
		}
	}

	for rtype, spec := range replicas {
		core.InitializeReplicaStatuses(status, rtype)

		err = r.ReconcilePods(ctx, job, status, pods, rtype, spec, replicas)
		if err != nil {
			logrus.Warnf(ErrReconcilePodsTemplate, err)
			return err
		}

		err = r.ReconcileServices(job, services, rtype, spec)
		if err != nil {
			logrus.Warnf(ErrReconcileServicesTemplate, err)
			return err
		}
	}

	err = r.UpdateJobStatus(job, replicas, status)
	if err != nil {
		logrus.Warnf(ErrUpdateJobErrorTemplate, err)
		return err
	}

	if !reflect.DeepEqual(*oldStatus, status) {
		return r.UpdateJobStatusInAPIServer(ctx, job)
	}

	return nil
}

// DeleteJob deletes this generic training job
func (r *JobReconciler) DeleteJob(job client.Object) error {
	return r.Delete(context.Background(), job)
}

// RecordAbnormalPods records abnormal pods during the reconciliation of jobs
func (r *JobReconciler) RecordAbnormalPods(activePods []*corev1.Pod, object client.Object) {
	core.RecordAbnormalPods(activePods, object, r.GetRecorder())
}

// SetStatusForSuccessJob sets the status for job that succeed
func (r *JobReconciler) SetStatusForSuccessJob(status *commonv1.JobStatus) {
	for rytpe := range status.ReplicaStatuses {
		status.ReplicaStatuses[rytpe].Succeeded += status.ReplicaStatuses[rytpe].Active
		status.ReplicaStatuses[rytpe].Active = 0
	}
}

// UpdateJobStatus updates the status of this generic training job WITHOUT pushing the updated status to the APIServer
func (r *JobReconciler) UpdateJobStatus(
	job client.Object,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	jobStatus *commonv1.JobStatus) error {

	logrus.Warnf(WarnDefaultImplementationTemplate, "UpdateJobStatus")

	jobKind := job.GetObjectKind().GroupVersionKind().Kind
	jobNamespacedName := types.NamespacedName{Namespace: job.GetNamespace(), Name: job.GetName()}.String()

	logger := r.GetLogger(job)

	for rtype, spec := range replicas {
		status, ok := jobStatus.ReplicaStatuses[rtype]
		if !ok {
			return fmt.Errorf(ErrGetReplicasStatusFromStatusFailedTemplate, rtype)
		}

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logrus.Infof("%s=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			jobKind, jobNamespacedName, rtype, expected, running, succeeded, failed)

		if r.IsFlagReplicaTypeForJobStatus(string(rtype)) {
			if running > 0 {
				msg := fmt.Sprintf("%s %s is running.", jobKind, jobNamespacedName)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg)
				if err != nil {
					logger.Info(ErrAppendJobConditionTemplate, err)
					return err
				}
			}

			if expected == 0 {
				msg := fmt.Sprintf("%s %s is successfully completed.", jobKind, jobNamespacedName)
				logrus.Info(msg)
				r.GetRecorder().Event(job, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, commonutil.JobSucceededReason, msg)
				if err != nil {
					logger.Info(ErrAppendJobConditionTemplate, err)
				}
				return nil
			}
		}

		if failed > 0 {
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				msg := fmt.Sprintf("%s %s is restarting because %d %s replica(s) failed.",
					jobKind, jobNamespacedName, failed, rtype)
				r.GetRecorder().Event(job, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					logger.Info(ErrAppendJobConditionTemplate, err)
					return err
				}
			} else {
				msg := fmt.Sprintf("%s %s is failed because %d %s replica(s) failed.",
					jobKind, jobNamespacedName, failed, rtype)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, commonutil.JobFailedReason, msg)
				if err != nil {
					logger.Info(ErrAppendJobConditionTemplate, err)
					return err
				}
			}
		}

	}

	msg := fmt.Sprintf("%s %s is running.", jobKind, jobNamespacedName)
	logger.Info(msg)

	if err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg); err != nil {
		logger.Error(err, ErrUpdateJobConditionsFailed, jobKind)
		return err
	}

	return nil
}

// UpdateJobStatusInAPIServer updates the status of this generic training job in APIServer
func (r *JobReconciler) UpdateJobStatusInAPIServer(ctx context.Context, job client.Object) error {
	return r.Status().Update(ctx, job)
}

// CleanupResources cleans up all resources associated with this generic training job
func (r *JobReconciler) CleanupResources(runPolicy *commonv1.RunPolicy, status commonv1.JobStatus, job client.Object) error {
	if *runPolicy.CleanPodPolicy == commonv1.CleanPodPolicyNone {
		return nil
	}
	ctx := context.Background()
	cleanRunningPod := *runPolicy.CleanPodPolicy == commonv1.CleanPodPolicyRunning

	if err := r.DeletePodGroup(ctx, job); err != nil {
		return err
	}

	pods, err := r.GetPodsForJob(ctx, job)
	if err != nil {
		return err
	}

	for _, pod := range pods {
		if cleanRunningPod && pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodPending {
			continue
		}
		if err = r.Delete(ctx, pod); err != nil {
			return err
		}
		// Each Pod may or may not has its service with same name
		svc := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, svc)
		if errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err = r.Delete(ctx, svc); err != nil {
			return err
		}

	}

	return nil
}

// CleanupJob cleans up all resources associated with this generic training job as well as the job itself
func (r *JobReconciler) CleanupJob(runPolicy *commonv1.RunPolicy, status commonv1.JobStatus, job client.Object) error {
	currentTime := time.Now()

	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		return nil
	}

	duration := time.Second * time.Duration(*ttl)
	// todo: Is the jobStatus.CompletionTime maybe nil ?
	finishTime := status.CompletionTime
	expireTime := finishTime.Add(duration)

	if currentTime.After(expireTime) {
		err := r.DeleteJob(job)
		if err != nil {
			commonutil.LoggerForJob(job).Warnf("Cleanup Job error: %v.", err)
			return err
		}
		return nil
	} else {
		if finishTime.After(currentTime) {
			commonutil.LoggerForJob(job).Warnf("Found Job finished in the future. This is likely due to time skew in the cluster. Job cleanup will be deferred.")
		}
	}
	return nil
}

// IsFlagReplicaTypeForJobStatus checks if this replicaType is the flag replicaType for the status of generic training job
func (r *JobReconciler) IsFlagReplicaTypeForJobStatus(rtype string) bool {
	logrus.Warnf(WarnDefaultImplementationTemplate, "IsFlagReplicaTypeForJobStatus")
	return true
}

// IsJobSucceeded checks if this generic training job succeeded
func (r *JobReconciler) IsJobSucceeded(status commonv1.JobStatus) bool {
	return commonutil.IsSucceeded(status)
}

// IsJobFailed checks if this generic training job failed
func (r *JobReconciler) IsJobFailed(status commonv1.JobStatus) bool {
	return commonutil.IsFailed(status)
}

// ShouldCleanUp checks if resources associated with this generic training job should be cleaned up
func (r *JobReconciler) ShouldCleanUp(status commonv1.JobStatus) bool {
	return r.IsJobSucceeded(status) || r.IsJobFailed(status)
}

// PastBackoffLimit checks if this generic training job has past backoff limit
func (r *JobReconciler) PastBackoffLimit(jobName string, runPolicy *commonv1.RunPolicy,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, pods []*corev1.Pod) (bool, error) {
	return core.PastBackoffLimit(jobName, runPolicy, replicas, pods, r.FilterPodsForReplicaType)
}

// PastActiveDeadline checks if this generic training job has ActiveDeadlineSeconds field set and if it is exceeded.
func (r *JobReconciler) PastActiveDeadline(runPolicy *commonv1.RunPolicy, jobStatus *commonv1.JobStatus) bool {
	return core.PastActiveDeadline(runPolicy, *jobStatus)
}

func (r *JobReconciler) GetJob(ctx context.Context, req ctrl.Request) (client.Object, error) {
	panic("implement me")
}

func (r *JobReconciler) ExtractReplicasSpec(job client.Object) (map[commonv1.ReplicaType]*commonv1.ReplicaSpec, error) {
	panic("implement me")
}

func (r *JobReconciler) ExtractRunPolicy(job client.Object) (*commonv1.RunPolicy, error) {
	panic("implement me")
}

func (r *JobReconciler) ExtractJobStatus(job client.Object) (*commonv1.JobStatus, error) {
	panic("implement me")
}

func (r *JobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	panic("implement me")
}
