package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestIsFinished(t *testing.T) {
	cases := map[string]struct {
		jobStatus apiv1.JobStatus
		want      bool
	}{
		"Succeeded job": {
			jobStatus: apiv1.JobStatus{
				Conditions: []apiv1.JobCondition{
					{
						Type:   apiv1.JobSucceeded,
						Status: corev1.ConditionTrue,
					},
				},
			},
			want: true,
		},
		"Failed job": {
			jobStatus: apiv1.JobStatus{
				Conditions: []apiv1.JobCondition{
					{
						Type:   apiv1.JobFailed,
						Status: corev1.ConditionTrue,
					},
				},
			},
			want: true,
		},
		"Suspended job": {
			jobStatus: apiv1.JobStatus{
				Conditions: []apiv1.JobCondition{
					{
						Type:   apiv1.JobSuspended,
						Status: corev1.ConditionTrue,
					},
				},
			},
			want: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsFinished(tc.jobStatus)
			if tc.want != got {
				t.Errorf("Unexpected result from IsFinished() \nwant: %v, got: %v\n", tc.want, got)
			}
		})
	}
}

func TestIsSucceeded(t *testing.T) {
	jobStatus := apiv1.JobStatus{
		Conditions: []apiv1.JobCondition{
			{
				Type:   apiv1.JobSucceeded,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, IsSucceeded(jobStatus))
}

func TestIsFailed(t *testing.T) {
	jobStatus := apiv1.JobStatus{
		Conditions: []apiv1.JobCondition{
			{
				Type:   apiv1.JobFailed,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, IsFailed(jobStatus))
}

func TestIsRunning(t *testing.T) {
	jobStatus := apiv1.JobStatus{
		Conditions: []apiv1.JobCondition{
			{
				Type:   apiv1.JobRunning,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, IsRunning(jobStatus))
}

func TestIsSuspended(t *testing.T) {
	jobStatus := apiv1.JobStatus{
		Conditions: []apiv1.JobCondition{
			{
				Type:   apiv1.JobSuspended,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, IsSuspended(jobStatus))
}

func TestUpdateJobConditions(t *testing.T) {
	jobStatus := apiv1.JobStatus{}
	conditionType := apiv1.JobCreated
	reason := "Job Created"
	message := "Job Created"

	err := UpdateJobConditions(&jobStatus, conditionType, corev1.ConditionTrue, reason, message)
	if assert.NoError(t, err) {
		// Check JobCreated condition is appended
		conditionInStatus := jobStatus.Conditions[0]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = apiv1.JobRunning
	reason = "Job Running"
	message = "Job Running"
	err = UpdateJobConditions(&jobStatus, conditionType, corev1.ConditionTrue, reason, message)
	if assert.NoError(t, err) {
		// Check JobRunning condition is appended
		conditionInStatus := jobStatus.Conditions[1]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = apiv1.JobRestarting
	reason = "Job Restarting"
	message = "Job Restarting"
	err = UpdateJobConditions(&jobStatus, conditionType, corev1.ConditionTrue, reason, message)
	if assert.NoError(t, err) {
		// Check JobRunning condition is filtered out and JobRestarting state is appended
		conditionInStatus := jobStatus.Conditions[1]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = apiv1.JobRunning
	reason = "Job Running"
	message = "Job Running"
	err = UpdateJobConditions(&jobStatus, conditionType, corev1.ConditionTrue, reason, message)
	if assert.NoError(t, err) {
		// Again, Check JobRestarting condition is filtered and JobRestarting is appended
		conditionInStatus := jobStatus.Conditions[1]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = apiv1.JobFailed
	reason = "Job Failed"
	message = "Job Failed"
	err = UpdateJobConditions(&jobStatus, conditionType, corev1.ConditionTrue, reason, message)
	if assert.NoError(t, err) {
		// Check JobRunning condition is set to false
		jobRunningCondition := jobStatus.Conditions[1]
		assert.Equal(t, jobRunningCondition.Type, apiv1.JobRunning)
		assert.Equal(t, jobRunningCondition.Status, corev1.ConditionFalse)
		// Check JobFailed state is appended
		conditionInStatus := jobStatus.Conditions[2]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}
}
