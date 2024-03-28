// Copyright 2018 The Kubeflow Authors
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

package train

import (
	"testing"

	"k8s.io/utils/ptr"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestIsRetryableExitCode(t *testing.T) {
	tcs := []struct {
		ExitCode int32
		Expected bool
	}{
		{
			ExitCode: 1,
			Expected: false,
		},
		{
			ExitCode: 2,
			Expected: false,
		},
		{
			ExitCode: 3,
			Expected: false,
		},
		{
			ExitCode: 130,
			Expected: true,
		},
		{
			ExitCode: 138,
			Expected: true,
		},
	}

	for _, tc := range tcs {
		actual := IsRetryableExitCode(tc.ExitCode)
		if actual != tc.Expected {
			t.Errorf("ExitCode %d: Expected %t, got %t", tc.ExitCode, tc.Expected, actual)
		}
	}
}

func TestIsJobSuspended(t *testing.T) {
	cases := map[string]struct {
		runPolicy *kubeflowv1.RunPolicy
		want      bool
	}{
		"runPolicy is nil": {
			runPolicy: nil,
			want:      false,
		},
		"suspend is nil": {
			runPolicy: &kubeflowv1.RunPolicy{
				Suspend: nil,
			},
			want: false,
		},
		"suspend is false": {
			runPolicy: &kubeflowv1.RunPolicy{
				Suspend: ptr.To(false),
			},
			want: false,
		},
		"suspend is true": {
			runPolicy: &kubeflowv1.RunPolicy{
				Suspend: ptr.To(true),
			},
			want: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsJobSuspended(tc.runPolicy)
			if tc.want != got {
				t.Errorf("Unexpected suspended from IsJobSuspended \nwant: %v\n, \ngot: %v\n", tc.want, got)
			}
		})
	}
}
