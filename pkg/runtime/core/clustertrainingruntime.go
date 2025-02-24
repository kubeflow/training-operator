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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	trainer "github.com/kubeflow/trainer/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/pkg/runtime"
)

var (
	errorNotFoundSpecifiedClusterTrainingRuntime = errors.New("not found ClusterTrainingRuntime specified in TrainJob")
)

type ClusterTrainingRuntime struct {
	*TrainingRuntime
}

var _ runtime.Runtime = (*ClusterTrainingRuntime)(nil)

var ClusterTrainingRuntimeGroupKind = schema.GroupKind{
	Group: trainer.GroupVersion.Group,
	Kind:  trainer.ClusterTrainingRuntimeKind,
}.String()

func NewClusterTrainingRuntime(context.Context, client.Client, client.FieldIndexer) (runtime.Runtime, error) {
	return &ClusterTrainingRuntime{
		TrainingRuntime: trainingRuntimeFactory,
	}, nil
}

func (r *ClusterTrainingRuntime) NewObjects(ctx context.Context, trainJob *trainer.TrainJob) ([]any, error) {
	var clTrainingRuntime trainer.ClusterTrainingRuntime
	if err := r.client.Get(ctx, client.ObjectKey{Name: trainJob.Spec.RuntimeRef.Name}, &clTrainingRuntime); err != nil {
		return nil, fmt.Errorf("%w: %w", errorNotFoundSpecifiedClusterTrainingRuntime, err)
	}
	return r.buildObjects(ctx, trainJob, clTrainingRuntime.Spec.Template, clTrainingRuntime.Spec.MLPolicy, clTrainingRuntime.Spec.PodGroupPolicy)
}

func (r *ClusterTrainingRuntime) TerminalCondition(ctx context.Context, trainJob *trainer.TrainJob) (*metav1.Condition, error) {
	return r.TrainingRuntime.TerminalCondition(ctx, trainJob)
}

func (r *ClusterTrainingRuntime) EventHandlerRegistrars() []runtime.ReconcilerBuilder {
	return nil
}

func (r *ClusterTrainingRuntime) ValidateObjects(ctx context.Context, old, new *trainer.TrainJob) (admission.Warnings, field.ErrorList) {
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: old.Namespace,
		Name:      old.Spec.RuntimeRef.Name,
	}, &trainer.ClusterTrainingRuntime{}); err != nil {
		return nil, field.ErrorList{
			field.Invalid(field.NewPath("spec", "RuntimeRef"), old.Spec.RuntimeRef,
				fmt.Sprintf("%v: specified clusterTrainingRuntime must be created before the TrainJob is created", err)),
		}
	}
	return r.framework.RunCustomValidationPlugins(old, new)
}
