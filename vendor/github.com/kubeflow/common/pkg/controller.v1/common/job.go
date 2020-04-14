package common

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	"github.com/kubeflow/common/pkg/util/k8sutil"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func (jc *JobController) deletePodsAndServices(runPolicy *apiv1.RunPolicy, job interface{}, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None.
	if *runPolicy.CleanPodPolicy == apiv1.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *runPolicy.CleanPodPolicy == apiv1.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
			continue
		}
		if err := jc.Controller.DeletePod(job, pod); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := jc.Controller.DeleteService(job, pod.Name, pod.Namespace); err != nil {
			return err
		}
	}
	return nil
}

func (jc *JobController) cleanupJobIfTTL(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus, job interface{}) error {
	currentTime := time.Now()
	metaObject, _ := job.(metav1.Object)
	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		// do nothing if the cleanup delay is not set
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(jobStatus.CompletionTime.Add(duration)) {
		err := jc.Controller.DeleteJob(job)
		if err != nil {
			commonutil.LoggerForJob(metaObject).Warnf("Cleanup Job error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(job)
	if err != nil {
		commonutil.LoggerForJob(metaObject).Warnf("Couldn't get key for job object: %v", err)
		return err
	}
	jc.WorkQueue.AddRateLimited(key)
	return nil
}

// ReconcileJobs checks and updates replicas for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (jc *JobController) ReconcileJobs(
	job interface{},
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec,
	jobStatus apiv1.JobStatus,
	runPolicy *apiv1.RunPolicy) error {

	metaObject, ok := job.(metav1.Object)
	jobName := metaObject.GetName()
	if !ok {
		return fmt.Errorf("job is not of type metav1.Object")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not of type runtime.Object")
	}
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for job object %#v: %v", job, err))
		return err
	}
	log.Infof("Reconciling for job %s", metaObject.GetName())

	oldStatus := jobStatus.DeepCopy()

	pods, err := jc.Controller.GetPodsForJob(job)

	if err != nil {
		log.Warnf("GetPodsForJob error %v", err)
		return err
	}

	services, err := jc.Controller.GetServicesForJob(job)

	if err != nil {
		log.Warnf("GetServicesForJob error %v", err)
		return err
	}

	// retrieve the previous number of retry
	previousRetry := jc.WorkQueue.NumRequeues(jobKey)

	activePods := k8sutil.FilterActivePods(pods)
	active := int32(len(activePods))
	failed := k8sutil.FilterPodCount(pods, v1.PodFailed)
	totalReplicas := k8sutil.GetTotalReplicas(replicas)
	prevReplicasFailedNum := k8sutil.GetTotalFailedReplicas(jobStatus.ReplicaStatuses)

	var failureMessage string
	jobExceedsLimit := false
	exceedsBackoffLimit := false
	pastBackoffLimit := false

	if runPolicy.BackoffLimit != nil {
		jobHasNewFailure := failed > prevReplicasFailedNum
		// new failures happen when status does not reflect the failures and active
		// is different than parallelism, otherwise the previous controller loop
		// failed updating status so even if we pick up failure it is not a new one
		exceedsBackoffLimit = jobHasNewFailure && (active != totalReplicas) &&
			(int32(previousRetry)+1 > *runPolicy.BackoffLimit)

		pastBackoffLimit, err = jc.pastBackoffLimit(jobName, runPolicy, replicas, pods)
		if err != nil {
			return err
		}
	}

	if exceedsBackoffLimit || pastBackoffLimit {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncJob
		jobExceedsLimit = true
		failureMessage = fmt.Sprintf("Job %s has failed because it has reached the specified backoff limit", jobName)
	} else if jc.pastActiveDeadline(runPolicy, jobStatus) {
		failureMessage = fmt.Sprintf("Job %s has failed because it was active longer than specified deadline", jobName)
		jobExceedsLimit = true
	}

	// If the Job is terminated, delete all pods and services.
	if commonutil.IsSucceeded(jobStatus) || commonutil.IsFailed(jobStatus) || jobExceedsLimit {
		if err := jc.deletePodsAndServices(runPolicy, job, pods); err != nil {
			return err
		}

		if err := jc.cleanupJob(runPolicy, jobStatus, job); err != nil {
			return err
		}

		if jc.Config.EnableGangScheduling {
			jc.Recorder.Event(runtimeObject, v1.EventTypeNormal, "JobTerminated", "Job has been terminated. Deleting PodGroup")
			if err := jc.DeletePodGroup(metaObject); err != nil {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeWarning, "FailedDeletePodGroup", "Error deleting: %v", err)
				return err
			} else {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, "SuccessfulDeletePodGroup", "Deleted PodGroup: %v", jobName)

			}
		}

		if jobExceedsLimit {
			jc.Recorder.Event(runtimeObject, v1.EventTypeNormal, commonutil.JobFailedReason, failureMessage)
			if jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}
			err := commonutil.UpdateJobConditions(&jobStatus, apiv1.JobFailed, commonutil.JobFailedReason, failureMessage)
			if err != nil {
				log.Infof("Append job condition error: %v", err)
				return err
			}
		}

		// At this point the pods may have been deleted.
		// 1) If the job succeeded, we manually set the replica status.
		// 2) If any replicas are still active, set their status to succeeded.
		if commonutil.IsSucceeded(jobStatus) {
			for rtype := range jobStatus.ReplicaStatuses {
				jobStatus.ReplicaStatuses[rtype].Succeeded += jobStatus.ReplicaStatuses[rtype].Active
				jobStatus.ReplicaStatuses[rtype].Active = 0
			}
		}
		return jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus)
	}

	// Save the current state of the replicas
	replicasStatus := make(map[string]v1.PodPhase)

	// Diff current active pods/services with replicas.
	for rtype, spec := range replicas {
		err := jc.ReconcilePods(metaObject, &jobStatus, pods, rtype, spec, replicasStatus, replicas)
		if err != nil {
			log.Warnf("ReconcilePods error %v", err)
			return err
		}

		err = jc.ReconcileServices(metaObject, services, rtype, spec)

		if err != nil {
			log.Warnf("ReconcileServices error %v", err)
			return err
		}
	}

	err = jc.Controller.UpdateJobStatus(job, replicas, &jobStatus)
	if err != nil {
		log.Warnf("UpdateJobStatus error %v", err)
		return err
	}
	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, jobStatus) {
		return jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus)
	}
	return nil
}

// pastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func (jc *JobController) pastActiveDeadline(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus) bool {
	if runPolicy.ActiveDeadlineSeconds == nil || jobStatus.StartTime == nil {
		return false
	}
	now := metav1.Now()
	start := jobStatus.StartTime.Time
	duration := now.Time.Sub(start)
	allowedDuration := time.Duration(*runPolicy.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

// pastBackoffLimit checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods with restartPolicy == OnFailure or Always
func (jc *JobController) pastBackoffLimit(jobName string, runPolicy *apiv1.RunPolicy,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, pods []*v1.Pod) (bool, error) {
	if runPolicy.BackoffLimit == nil {
		return false, nil
	}
	result := int32(0)
	for rtype, spec := range replicas {
		if spec.RestartPolicy != apiv1.RestartPolicyOnFailure && spec.RestartPolicy != apiv1.RestartPolicyAlways {
			log.Warnf("The restart policy of replica %v of the job %v is not OnFailure or Always. Not counted in backoff limit.", rtype, jobName)
			continue
		}
		// Convert ReplicaType to lower string.
		rt := strings.ToLower(string(rtype))
		pods, err := jc.FilterPodsForReplicaType(pods, rt)
		if err != nil {
			return false, err
		}
		for i := range pods {
			po := pods[i]
			if po.Status.Phase != v1.PodRunning {
				continue
			}
			for j := range po.Status.InitContainerStatuses {
				stat := po.Status.InitContainerStatuses[j]
				result += stat.RestartCount
			}
			for j := range po.Status.ContainerStatuses {
				stat := po.Status.ContainerStatuses[j]
				result += stat.RestartCount
			}
		}
	}

	if *runPolicy.BackoffLimit == 0 {
		return result > 0, nil
	}
	return result >= *runPolicy.BackoffLimit, nil
}

func (jc *JobController) cleanupJob(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus, job interface{}) error {
	currentTime := time.Now()
	metaObject, _ := job.(metav1.Object)
	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(jobStatus.CompletionTime.Add(duration)) {
		err := jc.Controller.DeleteJob(job)
		if err != nil {
			commonutil.LoggerForJob(metaObject).Warnf("Cleanup Job error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(job)
	if err != nil {
		commonutil.LoggerForJob(metaObject).Warnf("Couldn't get key for job object: %v", err)
		return err
	}
	jc.WorkQueue.AddRateLimited(key)
	return nil
}
