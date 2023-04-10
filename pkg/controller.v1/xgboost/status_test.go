package xgboost

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

var ignoreJobConditionsTimeOpts = cmpopts.IgnoreFields(commonv1.JobCondition{}, "LastUpdateTime", "LastTransitionTime")

func TestSetRunningCondition(t *testing.T) {
	jobName := "test-xbgoostjob"
	logger := logrus.NewEntry(logrus.New())
	tests := map[string]struct {
		input []commonv1.JobCondition
		want  []commonv1.JobCondition
	}{
		"input doesn't have a running condition": {
			input: []commonv1.JobCondition{
				{
					Type:    commonv1.JobSucceeded,
					Reason:  "XGBoostJobSucceeded",
					Message: "XGBoostJob test-xbgoostjob is successfully completed.",
					Status:  corev1.ConditionTrue,
				},
			},
			want: []commonv1.JobCondition{
				{
					Type:    commonv1.JobSucceeded,
					Reason:  "XGBoostJobSucceeded",
					Message: "XGBoostJob test-xbgoostjob is successfully completed.",
					Status:  corev1.ConditionTrue,
				},
				{
					Type:    commonv1.JobRunning,
					Reason:  "XGBoostJobRunning",
					Message: "XGBoostJob test-xbgoostjob is running.",
					Status:  corev1.ConditionTrue,
				},
			},
		},
		"input has a running condition": {
			input: []commonv1.JobCondition{
				{
					Type:    commonv1.JobFailed,
					Reason:  "XGBoostJobFailed",
					Message: "XGBoostJob test-sgboostjob is failed because 2 Worker replica(s) failed.",
					Status:  corev1.ConditionTrue,
				},
				{
					Type:    commonv1.JobRunning,
					Reason:  "XGBoostJobRunning",
					Message: "XGBoostJob test-xbgoostjob is running.",
					Status:  corev1.ConditionTrue,
				},
			},
			want: []commonv1.JobCondition{
				{
					Type:    commonv1.JobFailed,
					Reason:  "XGBoostJobFailed",
					Message: "XGBoostJob test-sgboostjob is failed because 2 Worker replica(s) failed.",
					Status:  corev1.ConditionTrue,
				},
				{
					Type:    commonv1.JobRunning,
					Reason:  "XGBoostJobRunning",
					Message: "XGBoostJob test-xbgoostjob is running.",
					Status:  corev1.ConditionTrue,
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jobStatus := &commonv1.JobStatus{Conditions: tc.input}
			err := setRunningCondition(logger, jobName, jobStatus)
			if err != nil {
				t.Fatalf("failed to update job condition: %v", err)
			}
			if diff := cmp.Diff(tc.want, jobStatus.Conditions, ignoreJobConditionsTimeOpts); len(diff) != 0 {
				t.Fatalf("Unexpected conditions from setRunningCondition (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestFindStatusCondition(t *testing.T) {
	tests := map[string]struct {
		conditions []commonv1.JobCondition
		want       *commonv1.JobCondition
	}{
		"conditions have a running condition": {
			conditions: []commonv1.JobCondition{
				{
					Type: commonv1.JobRunning,
				},
			},
			want: &commonv1.JobCondition{
				Type: commonv1.JobRunning,
			},
		},
		"condition doesn't have a running condition": {
			conditions: []commonv1.JobCondition{
				{
					Type: commonv1.JobSucceeded,
				},
			},
			want: nil,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := findStatusCondition(tc.conditions, commonv1.JobRunning)
			if diff := cmp.Diff(tc.want, got, ignoreJobConditionsTimeOpts); len(diff) != 0 {
				t.Fatalf("Unexpected jobConditions from findStatusCondition (-want,got):\n%s", diff)
			}
		})
	}
}
