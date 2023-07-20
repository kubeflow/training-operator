package util

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

const (
	// JobCreatedReason is added in a job when it is created.
	JobCreatedReason = "Created"
	// JobSucceededReason is added in a job when it is succeeded.
	JobSucceededReason = "Succeeded"
	// JobRunningReason is added in a job when it is running.
	JobRunningReason = "Running"
	// JobFailedReason is added in a job when it is failed.
	JobFailedReason = "Failed"
	// JobRestartingReason is added in a job when it is restarting.
	JobRestartingReason = "Restarting"
	// JobFailedValidationReason is added in a job when it failed validation
	JobFailedValidationReason = "FailedValidation"
	// JobSuspendedReason is added in a job when it is suspended.
	JobSuspendedReason = "Suspended"
	// JobResumedReason is added in a job when it is unsuspended.
	JobResumedReason = "Resumed"
)

func NewReason(kind, reason string) string {
	return fmt.Sprintf("%s%s", kind, reason)
}

// IsFinished checks if the job is succeeded or failed
func IsFinished(status apiv1.JobStatus) bool {
	return IsSucceeded(status) || IsFailed(status)
}

// IsSucceeded checks if the job is succeeded
func IsSucceeded(status apiv1.JobStatus) bool {
	return isStatusConditionTrue(status, apiv1.JobSucceeded)
}

// IsFailed checks if the job is failed
func IsFailed(status apiv1.JobStatus) bool {
	return isStatusConditionTrue(status, apiv1.JobFailed)
}

func IsRunning(status apiv1.JobStatus) bool {
	return isStatusConditionTrue(status, apiv1.JobRunning)
}

func IsSuspended(status apiv1.JobStatus) bool {
	return isStatusConditionTrue(status, apiv1.JobSuspended)
}

// UpdateJobConditions adds to the jobStatus a new condition if needed, with the conditionType, reason, and message
func UpdateJobConditions(
	jobStatus *apiv1.JobStatus,
	conditionType apiv1.JobConditionType,
	conditionStatus v1.ConditionStatus,
	reason, message string,
) error {
	condition := newCondition(conditionType, conditionStatus, reason, message)
	setCondition(jobStatus, condition)
	return nil
}

func isStatusConditionTrue(status apiv1.JobStatus, condType apiv1.JobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// newCondition creates a new job condition.
func newCondition(conditionType apiv1.JobConditionType, conditionStatus v1.ConditionStatus, reason, message string) apiv1.JobCondition {
	return apiv1.JobCondition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status apiv1.JobStatus, condType apiv1.JobConditionType) *apiv1.JobCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

// setCondition updates the job to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *apiv1.JobStatus, condition apiv1.JobCondition) {
	// Do nothing if JobStatus have failed condition
	if IsFailed(*status) {
		return
	}

	currentCond := getCondition(*status, condition.Type)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// Append the updated condition to the conditions
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of job conditions without conditions with the provided type.
func filterOutCondition(conditions []apiv1.JobCondition, condType apiv1.JobConditionType) []apiv1.JobCondition {
	var newConditions []apiv1.JobCondition
	for _, c := range conditions {
		if condType == apiv1.JobRestarting && c.Type == apiv1.JobRunning {
			continue
		}
		if condType == apiv1.JobRunning && c.Type == apiv1.JobRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if (condType == apiv1.JobFailed || condType == apiv1.JobSucceeded) && c.Type == apiv1.JobRunning {
			c.Status = v1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}
