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

// IsChieforMaster returns true if the type is Master or Chief.
func IsChieforMaster(typ TFReplicaType) bool {
	return typ == TFReplicaTypeChief || typ == TFReplicaTypeMaster
}

// IsWorker returns true if the type is Worker.
func IsWorker(typ TFReplicaType) bool {
	return typ == TFReplicaTypeWorker
}

// IsEvaluator returns true if the type is Evaluator.
func IsEvaluator(typ TFReplicaType) bool {
	return typ == TFReplicaTypeEval
}
