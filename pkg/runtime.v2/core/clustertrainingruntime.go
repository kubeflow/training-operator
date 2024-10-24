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

package core

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
)

var (
	errorNotFoundSpecifiedClusterTrainingRuntime = errors.New("not found ClusterTrainingRuntime specified in TrainJob")
)

type ClusterTrainingRuntime struct {
	*TrainingRuntime
}

var _ runtime.Runtime = (*ClusterTrainingRuntime)(nil)

var ClusterTrainingRuntimeGroupKind = schema.GroupKind{
	Group: kubeflowv2.GroupVersion.Group,
	Kind:  kubeflowv2.ClusterTrainingRuntimeKind,
}.String()

func NewClusterTrainingRuntime(context.Context, client.Client, client.FieldIndexer) (runtime.Runtime, error) {
	return &ClusterTrainingRuntime{
		TrainingRuntime: trainingRuntimeFactory,
	}, nil
}

func (r *ClusterTrainingRuntime) NewObjects(ctx context.Context, trainJob *kubeflowv2.TrainJob) ([]client.Object, error) {
	var clTrainingRuntime kubeflowv2.ClusterTrainingRuntime
	if err := r.client.Get(ctx, client.ObjectKey{Name: trainJob.Spec.RuntimeRef.Name}, &clTrainingRuntime); err != nil {
		return nil, fmt.Errorf("%w: %w", errorNotFoundSpecifiedClusterTrainingRuntime, err)
	}
	return r.buildObjects(ctx, trainJob, clTrainingRuntime.Spec.Template, clTrainingRuntime.Spec.MLPolicy, clTrainingRuntime.Spec.PodGroupPolicy)
}

func (r *ClusterTrainingRuntime) EventHandlerRegistrars() []runtime.ReconcilerBuilder {
	return nil
}

func (r *ClusterTrainingRuntime) ValidateObjects(ctx context.Context, old, new *kubeflowv2.TrainJob) (admission.Warnings, field.ErrorList) {
	clusterTrainingRuntime := &kubeflowv2.ClusterTrainingRuntime{}
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: new.Namespace,
		Name:      new.Spec.RuntimeRef.Name,
	}, clusterTrainingRuntime); err != nil {
		return nil, field.ErrorList{
			field.Invalid(field.NewPath("spec", "RuntimeRef"), new.Spec.RuntimeRef,
				fmt.Sprintf("%v: specified clusterTrainingRuntime must be created before the TrainJob is created", err)),
		}
	}
	info := r.getRuntimeInfo(ctx, new, clusterTrainingRuntime.Spec.Template, clusterTrainingRuntime.Spec.MLPolicy,
		clusterTrainingRuntime.Spec.PodGroupPolicy)
	return r.framework.RunCustomValidationPlugins(old, new, info)
}
