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

package webhookv2

import (
	ctrl "sigs.k8s.io/controller-runtime"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
)

func Setup(mgr ctrl.Manager, runtimes map[string]runtime.Runtime) (string, error) {
	if err := setupWebhookForClusterTrainingRuntime(mgr, runtimes); err != nil {
		return kubeflowv2.ClusterTrainingRuntimeKind, err
	}
	if err := setupWebhookForTrainingRuntime(mgr, runtimes); err != nil {
		return kubeflowv2.TrainingRuntimeKind, err
	}
	if err := setupWebhookForTrainJob(mgr, runtimes); err != nil {
		return kubeflowv2.TrainJobKind, err
	}
	return "", nil
}
