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

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

func TestDurationUntilExpireTime(t *testing.T) {
	tests := []struct {
		name      string
		runPolicy *commonv1.RunPolicy
		jobStatus commonv1.JobStatus
		want      time.Duration
		wantErr   bool
	}{
		{
			name:      "running job",
			runPolicy: &commonv1.RunPolicy{},
			jobStatus: commonv1.JobStatus{
				Conditions: []commonv1.JobCondition{newJobCondition(commonv1.JobRunning)},
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "succeeded job with remaining time 1s",
			runPolicy: &commonv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: commonv1.JobStatus{
				Conditions:     []commonv1.JobCondition{newJobCondition(commonv1.JobSucceeded)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(4 * time.Second)},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "failed job with remaining time 1s",
			runPolicy: &commonv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: commonv1.JobStatus{
				Conditions:     []commonv1.JobCondition{newJobCondition(commonv1.JobFailed)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(4 * time.Second)},
			},
			want:    1,
			wantErr: false,
		},
		{
			name:      "succeeded job with infinite TTL",
			runPolicy: &commonv1.RunPolicy{},
			jobStatus: commonv1.JobStatus{
				Conditions:     []commonv1.JobCondition{newJobCondition(commonv1.JobSucceeded)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(4 * time.Second)},
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "succeeded job without remaining time",
			runPolicy: &commonv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: commonv1.JobStatus{
				Conditions:     []commonv1.JobCondition{newJobCondition(commonv1.JobSucceeded)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(6 * time.Second)},
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "succeeded job with nil completion time error",
			runPolicy: &commonv1.RunPolicy{
				TTLSecondsAfterFinished: pointer.Int32(5),
			},
			jobStatus: commonv1.JobStatus{
				Conditions: []commonv1.JobCondition{newJobCondition(commonv1.JobSucceeded)},
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

func newJobCondition(t commonv1.JobConditionType) commonv1.JobCondition {
	return commonv1.JobCondition{
		Type:   t,
		Status: corev1.ConditionTrue,
	}
}
