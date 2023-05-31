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

package common_test

import (
	"testing"

	commonv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"

	"github.com/kubeflow/training-operator/test_job/reconciler.v1/test_job"
)

func TestGenLabels(t *testing.T) {
	type tc struct {
		testJobName   string
		expectedLabel map[string]string
	}
	testCase := []tc{
		func() tc {
			return tc{
				testJobName: "test/job1",
				expectedLabel: map[string]string{
					commonv1.JobNameLabel:      "test-job1",
					commonv1.OperatorNameLabel: "Test Reconciler",
				},
			}
		}(),
	}

	testReconciler := test_job.NewTestReconciler()

	for _, c := range testCase {
		labels := testReconciler.GenLabels(c.testJobName)
		if len(labels) != len(c.expectedLabel) {
			t.Errorf("Expected to get %d labels, got %d labels", len(c.expectedLabel), len(labels))
			continue
		}
		for ek, ev := range c.expectedLabel {
			if v, ok := labels[ek]; !ok {
				t.Errorf("Cannot found expected key %s", ek)
			} else {
				if ev != v {
					t.Errorf("Expected to get %s for %s, got %s", ev, ek, v)
				}
			}
		}
	}
}
