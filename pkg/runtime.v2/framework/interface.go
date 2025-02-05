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

package framework

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/trainer/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
)

type Plugin interface {
	Name() string
}

type WatchExtensionPlugin interface {
	Plugin
	ReconcilerBuilders() []runtime.ReconcilerBuilder
}

type EnforcePodGroupPolicyPlugin interface {
	Plugin
	EnforcePodGroupPolicy(info *runtime.Info, trainJob *kubeflowv2.TrainJob) error
}

type EnforceMLPolicyPlugin interface {
	Plugin
	EnforceMLPolicy(info *runtime.Info, trainJob *kubeflowv2.TrainJob) error
}

type CustomValidationPlugin interface {
	Plugin
	Validate(oldObj, newObj *kubeflowv2.TrainJob) (admission.Warnings, field.ErrorList)
}

type ComponentBuilderPlugin interface {
	Plugin
	Build(ctx context.Context, runtimeJobTemplate client.Object, info *runtime.Info, trainJob *kubeflowv2.TrainJob) (client.Object, error)
}

type TerminalConditionPlugin interface {
	Plugin
	TerminalCondition(ctx context.Context, trainJob *kubeflowv2.TrainJob) (*metav1.Condition, error)
}
