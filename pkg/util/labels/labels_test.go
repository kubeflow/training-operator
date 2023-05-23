/*
Copyright 2023 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package labels

import (
	"testing"

	v1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestReplicaIndex(t *testing.T) {
	cases := map[string]struct {
		labels  map[string]string
		want    int
		wantErr bool
	}{
		"new": {
			labels: map[string]string{
				v1.ReplicaIndexLabel: "2",
			},
			want: 2,
		},
		"old": {
			labels: map[string]string{
				v1.ReplicaIndexLabel: "3",
			},
			want: 3,
		},
		"none": {
			labels:  map[string]string{},
			wantErr: true,
		},
		"both": {
			labels: map[string]string{
				v1.ReplicaIndexLabel: "4",
			},
			want: 4,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ReplicaIndex(tc.labels)
			if gotErr := err != nil; tc.wantErr != gotErr {
				t.Errorf("ReplicaIndex returned error (%t) want (%t)", gotErr, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ReplicaIndex returned %d, want %d", got, tc.want)
			}
		})
	}
}

func TestReplicaType(t *testing.T) {
	cases := map[string]struct {
		labels  map[string]string
		want    v1.ReplicaType
		wantErr bool
	}{
		"new": {
			labels: map[string]string{
				v1.ReplicaTypeLabel: "Foo",
			},
			want: "Foo",
		},
		"old": {
			labels: map[string]string{
				v1.ReplicaTypeLabel: "Bar",
			},
			want: "Bar",
		},
		"none": {
			labels:  map[string]string{},
			wantErr: true,
		},
		"both": {
			labels: map[string]string{
				v1.ReplicaTypeLabel: "Baz",
			},
			want: "Baz",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ReplicaType(tc.labels)
			if gotErr := err != nil; tc.wantErr != gotErr {
				t.Errorf("ReplicaType returned error (%t) want (%t)", gotErr, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ReplicaType returned %v, want %v", got, tc.want)
			}
		})
	}
}
