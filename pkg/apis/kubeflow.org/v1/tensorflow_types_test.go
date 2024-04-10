/*
Copyright 2024 The Kubeflow Authors.

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

package v1

import "testing"

func TestIsChiefOrMaster(t *testing.T) {
	tc := []struct {
		Type     ReplicaType
		Expected bool
	}{
		{
			Type:     TFJobReplicaTypeChief,
			Expected: true,
		},
		{
			Type:     TFJobReplicaTypeMaster,
			Expected: true,
		},
		{
			Type:     TFJobReplicaTypeWorker,
			Expected: false,
		},
	}
	for _, c := range tc {
		actual := IsChiefOrMaster(c.Type)
		if actual != c.Expected {
			t.Errorf("Expected %v; Got %v", c.Expected, actual)
		}
	}
}
