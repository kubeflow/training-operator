package xgboost

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
)

var ignoreJobConditionsTimeOpts = cmpopts.IgnoreFields(kubeflowv1.JobCondition{}, "LastUpdateTime", "LastTransitionTime")

func TestSetRunningCondition(t *testing.T) {
	jobName := "test-xbgoostjob"
	logger := logrus.NewEntry(logrus.New())
	tests := map[string]struct {
		input []kubeflowv1.JobCondition
		want  []kubeflowv1.JobCondition
	}{
		"input doesn't have a running condition": {
			input: []kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobSucceeded,
					Reason:  commonutil.NewReason(kubeflowv1.XGBoostJobKind, commonutil.JobSucceededReason),
					Message: "XGBoostJob test-xbgoostjob is successfully completed.",
					Status:  corev1.ConditionTrue,
				},
			},
			want: []kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobSucceeded,
					Reason:  commonutil.NewReason(kubeflowv1.XGBoostJobKind, commonutil.JobSucceededReason),
					Message: "XGBoostJob test-xbgoostjob is successfully completed.",
					Status:  corev1.ConditionTrue,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Reason:  commonutil.NewReason(kubeflowv1.XGBoostJobKind, commonutil.JobRunningReason),
					Message: "XGBoostJob test-xbgoostjob is running.",
					Status:  corev1.ConditionTrue,
				},
			},
		},
		"input has a running condition": {
			input: []kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobFailed,
					Reason:  commonutil.NewReason(kubeflowv1.XGBoostJobKind, commonutil.JobFailedReason),
					Message: "XGBoostJob test-sgboostjob is failed because 2 Worker replica(s) failed.",
					Status:  corev1.ConditionTrue,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Reason:  commonutil.NewReason(kubeflowv1.XGBoostJobKind, commonutil.JobRunningReason),
					Message: "XGBoostJob test-xbgoostjob is running.",
					Status:  corev1.ConditionTrue,
				},
			},
			want: []kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobFailed,
					Reason:  commonutil.NewReason(kubeflowv1.XGBoostJobKind, commonutil.JobFailedReason),
					Message: "XGBoostJob test-sgboostjob is failed because 2 Worker replica(s) failed.",
					Status:  corev1.ConditionTrue,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Reason:  commonutil.NewReason(kubeflowv1.XGBoostJobKind, commonutil.JobRunningReason),
					Message: "XGBoostJob test-xbgoostjob is running.",
					Status:  corev1.ConditionTrue,
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jobStatus := &kubeflowv1.JobStatus{Conditions: tc.input}
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
		conditions []kubeflowv1.JobCondition
		want       *kubeflowv1.JobCondition
	}{
		"conditions have a running condition": {
			conditions: []kubeflowv1.JobCondition{
				{
					Type: kubeflowv1.JobRunning,
				},
			},
			want: &kubeflowv1.JobCondition{
				Type: kubeflowv1.JobRunning,
			},
		},
		"condition doesn't have a running condition": {
			conditions: []kubeflowv1.JobCondition{
				{
					Type: kubeflowv1.JobSucceeded,
				},
			},
			want: nil,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := findStatusCondition(tc.conditions, kubeflowv1.JobRunning)
			if diff := cmp.Diff(tc.want, got, ignoreJobConditionsTimeOpts); len(diff) != 0 {
				t.Fatalf("Unexpected jobConditions from findStatusCondition (-want,got):\n%s", diff)
			}
		})
	}
}
