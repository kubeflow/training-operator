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

package v1

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

func TestIsChieforMaster(t *testing.T) {
	tc := []struct {
		Type     commonv1.ReplicaType
		Expected bool
	}{
		{
			Type:     TFReplicaTypeChief,
			Expected: true,
		},
		{
			Type:     TFReplicaTypeMaster,
			Expected: true,
		},
		{
			Type:     TFReplicaTypeWorker,
			Expected: false,
		},
	}

	for _, c := range tc {
		actual := IsChieforMaster(c.Type)
		if actual != c.Expected {
			t.Errorf("Expected %v; Got %v", c.Expected, actual)
		}
	}
}
