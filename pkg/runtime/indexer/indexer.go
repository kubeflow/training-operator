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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/trainer/pkg/apis/trainer/v1alpha1"
)

const (
	TrainJobRuntimeRefKey        = ".spec.runtimeRef.kind=trainingRuntime"
	TrainJobClusterRuntimeRefKey = ".spec.runtimeRef.kind=clusterTrainingRuntime"
)

func IndexTrainJobTrainingRuntime(obj client.Object) []string {
	trainJob, ok := obj.(*kubeflowv1.TrainJob)
	if !ok {
		return nil
	}
	if ptr.Deref(trainJob.Spec.RuntimeRef.APIGroup, "") == kubeflowv1.GroupVersion.Group &&
		ptr.Deref(trainJob.Spec.RuntimeRef.Kind, "") == kubeflowv1.TrainingRuntimeKind {
		return []string{trainJob.Spec.RuntimeRef.Name}
	}
	return nil
}

func IndexTrainJobClusterTrainingRuntime(obj client.Object) []string {
	trainJob, ok := obj.(*kubeflowv1.TrainJob)
	if !ok {
		return nil
	}
	if ptr.Deref(trainJob.Spec.RuntimeRef.APIGroup, "") == kubeflowv1.GroupVersion.Group &&
		ptr.Deref(trainJob.Spec.RuntimeRef.Kind, "") == kubeflowv1.ClusterTrainingRuntimeKind {
		return []string{trainJob.Spec.RuntimeRef.Name}
	}
	return nil
}
