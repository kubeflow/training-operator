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

package indexer

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
)

const (
	TrainJobTrainingRuntimeRefKey = ".spec.trainingRuntimeRef"
)

func IndexTrainJobTrainingRuntimes(obj client.Object) []string {
	trainJob, ok := obj.(*kubeflowv2.TrainJob)
	if !ok {
		return nil
	}
	runtimeRefGroupKind := schema.GroupKind{
		Group: ptr.Deref(trainJob.Spec.TrainingRuntimeRef.APIGroup, ""),
		Kind:  ptr.Deref(trainJob.Spec.TrainingRuntimeRef.Kind, ""),
	}
	if runtimeRefGroupKind.Group == kubeflowv2.GroupVersion.Group &&
		(runtimeRefGroupKind.Kind == "TrainingRuntime" || runtimeRefGroupKind.Kind == "ClusterTrainingRuntime") {
		return []string{trainJob.Spec.TrainingRuntimeRef.Name}
	}
	return nil
}
