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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	fwkcore "github.com/kubeflow/training-operator/pkg/runtime.v2/framework/core"
	fwkplugins "github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins"
	idxer "github.com/kubeflow/training-operator/pkg/runtime.v2/indexer"
)

var (
	errorNotFoundSpecifiedTrainingRuntime = errors.New("TrainingRuntime specified in TrainJob is not found")
)

type TrainingRuntime struct {
	framework *fwkcore.Framework
	client    client.Client
}

var TrainingRuntimeGroupKind = schema.GroupKind{
	Group: kubeflowv2.GroupVersion.Group,
	Kind:  kubeflowv2.TrainingRuntimeKind,
}.String()

var _ runtime.Runtime = (*TrainingRuntime)(nil)

var trainingRuntimeFactory *TrainingRuntime

func NewTrainingRuntime(ctx context.Context, c client.Client, indexer client.FieldIndexer) (runtime.Runtime, error) {
	if err := indexer.IndexField(ctx, &kubeflowv2.TrainJob{}, idxer.TrainJobTrainingRuntimeRefKey, idxer.IndexTrainJobTrainingRuntimes); err != nil {
		return nil, fmt.Errorf("setting index on TrainingRuntime and ClusterTrainigRuntime for TrainJob: %w", err)
	}
	fwk, err := fwkcore.New(ctx, c, fwkplugins.NewRegistry(), indexer)
	if err != nil {
		return nil, err
	}
	trainingRuntimeFactory = &TrainingRuntime{
		framework: fwk,
		client:    c,
	}
	return trainingRuntimeFactory, nil
}

func (r *TrainingRuntime) NewObjects(ctx context.Context, trainJob *kubeflowv2.TrainJob) ([]client.Object, error) {
	var trainingRuntime kubeflowv2.TrainingRuntime
	err := r.client.Get(ctx, client.ObjectKey{Namespace: trainJob.Namespace, Name: trainJob.Spec.TrainingRuntimeRef.Name}, &trainingRuntime)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errorNotFoundSpecifiedTrainingRuntime, err)
	}
	return r.buildObjects(ctx, trainJob, trainingRuntime.Spec.Template, trainingRuntime.Spec.MLPolicy, trainingRuntime.Spec.PodGroupPolicy)
}

func (r *TrainingRuntime) buildObjects(ctx context.Context, trainJob *kubeflowv2.TrainJob, jobSetTemplateSpec kubeflowv2.JobSetTemplateSpec,
	mlPolicy *kubeflowv2.MLPolicy, podGroupPolicy *kubeflowv2.PodGroupPolicy) ([]client.Object, error) {
	opts := []runtime.InfoOption{
		runtime.WithLabels(jobSetTemplateSpec.Labels),
		runtime.WithAnnotations(jobSetTemplateSpec.Annotations),
		runtime.WithMLPolicy(mlPolicy),
		runtime.WithPodGroupPolicy(podGroupPolicy),
	}
	for idx, rJob := range jobSetTemplateSpec.Spec.ReplicatedJobs {
		if rJob.Replicas == 0 {
			jobSetTemplateSpec.Spec.ReplicatedJobs[idx].Replicas = 1
		}
		replicas := jobSetTemplateSpec.Spec.ReplicatedJobs[idx].Replicas * ptr.Deref(rJob.Template.Spec.Completions, 1)
		opts = append(opts, runtime.WithPodSpecReplicas(rJob.Name, replicas, rJob.Template.Spec.Template.Spec))
	}
	info := runtime.NewInfo(&jobsetv1alpha2.JobSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: jobsetv1alpha2.SchemeGroupVersion.String(),
			Kind:       "JobSet",
		},
		Spec: *jobSetTemplateSpec.Spec.DeepCopy(),
	}, opts...)

	if err := r.framework.RunEnforceMLPolicyPlugins(info); err != nil {
		return nil, err
	}
	err := r.framework.RunEnforcePodGroupPolicyPlugins(trainJob, info)
	if err != nil {
		return nil, err
	}
	return r.framework.RunComponentBuilderPlugins(ctx, info, trainJob)
}

func (r *TrainingRuntime) EventHandlerRegistrars() []runtime.ReconcilerBuilder {
	var builders []runtime.ReconcilerBuilder
	for _, ex := range r.framework.WatchExtensionPlugins() {
		builders = append(builders, ex.ReconcilerBuilders()...)
	}
	return builders
}

func (r *TrainingRuntime) ValidateObjects(ctx context.Context, old, new *kubeflowv2.TrainJob) (admission.Warnings, field.ErrorList) {
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: old.Namespace,
		Name:      old.Spec.TrainingRuntimeRef.Name,
	}, &kubeflowv2.TrainingRuntime{}); err != nil {
		return nil, field.ErrorList{
			field.Invalid(field.NewPath("spec", "trainingRuntimeRef"), old.Spec.TrainingRuntimeRef,
				fmt.Sprintf("%v: specified trainingRuntime must be created before the TrainJob is created", err)),
		}
	}
	return r.framework.RunCustomValidationPlugins(old, new)
}
