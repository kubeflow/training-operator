// Copyright 2022 The Kubeflow Authors
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

package util

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestDurationUntilExpireTime(t *testing.T) {
	tests := []struct {
		name      string
		runPolicy *kubeflowv1.RunPolicy
		jobStatus kubeflowv1.JobStatus
		want      time.Duration
		wantErr   bool
	}{
		{
			name:      "running job",
			runPolicy: &kubeflowv1.RunPolicy{},
			jobStatus: kubeflowv1.JobStatus{
				Conditions: []kubeflowv1.JobCondition{newJobCondition(kubeflowv1.JobRunning)},
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "succeeded job with remaining time 1s",
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: kubeflowv1.JobStatus{
				Conditions:     []kubeflowv1.JobCondition{newJobCondition(kubeflowv1.JobSucceeded)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(4 * time.Second)},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "failed job with remaining time 1s",
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: kubeflowv1.JobStatus{
				Conditions:     []kubeflowv1.JobCondition{newJobCondition(kubeflowv1.JobFailed)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(4 * time.Second)},
			},
			want:    1,
			wantErr: false,
		},
		{
			name:      "succeeded job with infinite TTL",
			runPolicy: &kubeflowv1.RunPolicy{},
			jobStatus: kubeflowv1.JobStatus{
				Conditions:     []kubeflowv1.JobCondition{newJobCondition(kubeflowv1.JobSucceeded)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(4 * time.Second)},
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "succeeded job without remaining time",
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: kubeflowv1.JobStatus{
				Conditions:     []kubeflowv1.JobCondition{newJobCondition(kubeflowv1.JobSucceeded)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(6 * time.Second)},
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "succeeded job with nil completion time error",
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: kubeflowv1.JobStatus{
				Conditions: []kubeflowv1.JobCondition{newJobCondition(kubeflowv1.JobSucceeded)},
			},
			want:    -1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DurationUntilExpireTime(tt.runPolicy, tt.jobStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("DurationUntilExpireTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				if tt.want < 0 || tt.want >= 0 && tt.want > got {
					t.Errorf("DurationUntilExpireTime() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func newJobCondition(t kubeflowv1.JobConditionType) kubeflowv1.JobCondition {
	return kubeflowv1.JobCondition{
		Type:   t,
		Status: corev1.ConditionTrue,
	}
}
