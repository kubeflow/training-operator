// Copyright 2020 The Kubeflow Authors
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

// Package controller provides a Kubernetes controller for a TFJob resource.
package tensorflow

import (
	"reflect"
	"testing"
)

func TestConvertClusterSpecToSparseClusterSpec(t *testing.T) {
	clusterSpec := ClusterSpec{
		"ps":     {"test-tfjob-ps-0.default.svc:2222", "test-tfjob-ps-1.default.svc:2222"},
		"worker": {"test-tfjob-worker-0.default.svc:2222", "test-tfjob-worker-1.default.svc:2222"},
	}
	workerSparseClusterSpec := convertClusterSpecToSparseClusterSpec(clusterSpec, "worker", 0)
	psSparseClusterSpec := convertClusterSpecToSparseClusterSpec(clusterSpec, "ps", 0)

	expectedWorkerSparseClusterSpec := SparseClusterSpec{
		Worker: map[int32]string{0: "test-tfjob-worker-0.default.svc:2222"},
		PS:     []string{"test-tfjob-ps-0.default.svc:2222", "test-tfjob-ps-1.default.svc:2222"},
	}
	expectedPSSparseClusterSpec := SparseClusterSpec{
		Worker: map[int32]string{},
		PS:     []string{"test-tfjob-ps-0.default.svc:2222"},
	}
	if !reflect.DeepEqual(workerSparseClusterSpec, expectedWorkerSparseClusterSpec) {
		t.Error("sparseClusterSpec for worker is not correct!")
	}
	if !reflect.DeepEqual(psSparseClusterSpec, expectedPSSparseClusterSpec) {
		t.Error("sparseClusterSpec for worker is not correct!")
	}
}
