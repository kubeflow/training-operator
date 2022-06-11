// Copyright 2019 The Kubeflow Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mpi

import (
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/training/v1"
)

const (
	configSuffix            = "-config"
	configVolumeName        = "mpi-job-config"
	configMountPath         = "/etc/mpi"
	kubexecScriptName       = "kubexec.sh"
	hostfileName            = "hostfile"
	discoverHostsScriptName = "discover_hosts.sh"
	kubectlDeliveryName     = "kubectl-delivery"
	kubectlTargetDirEnv     = "TARGET_DIR"
	kubectlVolumeName       = "mpi-job-kubectl"
	kubectlMountPath        = "/opt/kube"
	launcher                = "launcher"
	worker                  = "worker"
	launcherSuffix          = "-launcher"
	workerSuffix            = "-worker"
	gpuResourceNameSuffix   = ".com/gpu"
	gpuResourceNamePattern  = "gpu"
	initContainerCpu        = "100m"
	initContainerEphStorage = "5Gi"
	initContainerMem        = "512Mi"
)

const (
	// ErrResourceExists is used as part of the Event 'reason' when an MPIJob
	// fails to sync due to dependent resources of the same name already
	// existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to dependent resources already existing.
	MessageResourceExists = "Resource %q of Kind %q already exists and is not managed by MPIJob"

	// ErrResourceDoesNotExist is used as part of the Event 'reason' when some
	// resource is missing in yaml
	ErrResourceDoesNotExist = "ErrResourceDoesNotExist"

	// MessageResourceDoesNotExist is used for Events when some
	// resource is missing in yaml
	MessageResourceDoesNotExist = "Resource %q is missing in yaml"

	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"

	// gang scheduler name.
	gangSchedulerName = "volcano"

	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
	// gangSchedulingPodGroupAnnotation is the annotation key used by batch schedulers
	gangSchedulingPodGroupAnnotation = "scheduling.k8s.io/group-name"

	// volcanoTaskSpecKey task spec key used in pod annotation when EnableGangScheduling is true
	volcanoTaskSpecKey = "volcano.sh/task-spec"
)

const (
	// mpiJobCreatedReason is added in a mpijob when it is created.
	mpiJobCreatedReason = "MPIJobCreated"
	// mpiJobSucceededReason is added in a mpijob when it is succeeded.
	mpiJobSucceededReason = "MPIJobSucceeded"
	// mpiJobRunningReason is added in a mpijob when it is running.
	mpiJobRunningReason = "MPIJobRunning"
	// mpiJobFailedReason is added in a mpijob when it is failed.
	mpiJobFailedReason = "MPIJobFailed"
	// mpiJobEvict
	mpiJobEvict = "MPIJobEvicted"
)

// initializeMPIJobStatuses initializes the ReplicaStatuses for MPIJob.
func initializeMPIJobStatuses(mpiJob *trainingv1.MPIJob, mtype commonv1.ReplicaType) {
	replicaType := commonv1.ReplicaType(mtype)
	if mpiJob.Status.ReplicaStatuses == nil {
		mpiJob.Status.ReplicaStatuses = make(map[commonv1.ReplicaType]*commonv1.ReplicaStatus)
	}

	mpiJob.Status.ReplicaStatuses[replicaType] = &commonv1.ReplicaStatus{}
}

// updateMPIJobConditions updates the conditions of the given mpiJob.
func updateMPIJobConditions(mpiJob *trainingv1.MPIJob, conditionType commonv1.JobConditionType, reason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(&mpiJob.Status, condition)
	return nil
}

// newCondition creates a new mpiJob condition.
func newCondition(conditionType commonv1.JobConditionType, reason, message string) commonv1.JobCondition {
	return commonv1.JobCondition{
		Type:               conditionType,
		Status:             corev1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status commonv1.JobStatus, condType commonv1.JobConditionType) *commonv1.JobCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

func isEvicted(status commonv1.JobStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type == commonv1.JobFailed &&
			condition.Status == corev1.ConditionTrue &&
			condition.Reason == mpiJobEvict {
			return true
		}
	}
	return false
}

// setCondition updates the mpiJob to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *commonv1.JobStatus, condition commonv1.JobCondition) {

	currentCond := getCondition(*status, condition.Type)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// Append the updated condition
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of mpiJob conditions without conditions with the provided type.
func filterOutCondition(conditions []commonv1.JobCondition, condType commonv1.JobConditionType) []commonv1.JobCondition {
	var newConditions []commonv1.JobCondition
	for _, c := range conditions {
		if condType == commonv1.JobRestarting && c.Type == commonv1.JobRunning {
			continue
		}
		if condType == commonv1.JobRunning && c.Type == commonv1.JobRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if (condType == commonv1.JobFailed || condType == commonv1.JobSucceeded) && (c.Type == commonv1.JobRunning || c.Type == commonv1.JobFailed) {
			c.Status = corev1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}

func isPodFinished(j *corev1.Pod) bool {
	return isPodSucceeded(j) || isPodFailed(j)
}

func isPodFailed(p *corev1.Pod) bool {
	return p.Status.Phase == corev1.PodFailed
}

func isPodSucceeded(p *corev1.Pod) bool {
	return p.Status.Phase == corev1.PodSucceeded
}

func isPodRunning(p *corev1.Pod) bool {
	return p.Status.Phase == corev1.PodRunning
}

// isGPULauncher checks whether the launcher needs GPU.
func isGPULauncher(mpiJob *trainingv1.MPIJob) bool {
	for _, container := range mpiJob.Spec.MPIReplicaSpecs[trainingv1.MPIReplicaTypeLauncher].Template.Spec.Containers {
		for key := range container.Resources.Limits {
			if strings.HasSuffix(string(key), gpuResourceNameSuffix) {
				return true
			}
			if strings.Contains(string(key), gpuResourceNamePattern) {
				return true
			}
		}
	}
	return false
}

func defaultReplicaLabels(genericLabels map[string]string, roleLabelVal string) map[string]string {
	replicaLabels := map[string]string{}
	for k, v := range genericLabels {
		replicaLabels[k] = v
	}

	replicaLabels[commonv1.ReplicaTypeLabel] = roleLabelVal
	return replicaLabels
}

func defaultWorkerLabels(genericLabels map[string]string) map[string]string {
	return defaultReplicaLabels(genericLabels, worker)
}

func defaultLauncherLabels(genericLabels map[string]string) map[string]string {
	return defaultReplicaLabels(genericLabels, launcher)
}

func workerSelector(genericLabels map[string]string) (labels.Selector, error) {
	labels := defaultWorkerLabels(genericLabels)

	labelSelector := metav1.LabelSelector{
		MatchLabels: labels,
	}

	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return nil, err
	}

	return selector, nil
}

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
// originally from pkg/controller.v1/tensorflow/status.go (deleted)
func initializeReplicaStatuses(jobStatus *commonv1.JobStatus, rtype commonv1.ReplicaType) {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = make(map[commonv1.ReplicaType]*commonv1.ReplicaStatus)
	}

	jobStatus.ReplicaStatuses[rtype] = &commonv1.ReplicaStatus{}
}
