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

package jobset

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	"github.com/kubeflow/training-operator/pkg/constants"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
)

type JobSet struct {
	client     client.Client
	restMapper meta.RESTMapper
	scheme     *apiruntime.Scheme
	logger     logr.Logger
}

var _ framework.WatchExtensionPlugin = (*JobSet)(nil)
var _ framework.ComponentBuilderPlugin = (*JobSet)(nil)
var _ framework.TerminalConditionPlugin = (*JobSet)(nil)
var _ framework.CustomValidationPlugin = (*JobSet)(nil)

const Name = constants.JobSetKind

//+kubebuilder:rbac:groups=jobset.x-k8s.io,resources=jobsets,verbs=get;list;watch;create

func New(ctx context.Context, c client.Client, _ client.FieldIndexer) (framework.Plugin, error) {
	return &JobSet{
		client:     c,
		restMapper: c.RESTMapper(),
		scheme:     c.Scheme(),
		logger:     ctrl.LoggerFrom(ctx).WithValues("pluginName", constants.JobSetKind),
	}, nil
}

func (j *JobSet) Name() string {
	return Name
}

func (j *JobSet) Build(ctx context.Context, runtimeJobTemplate client.Object, info *runtime.Info, trainJob *kubeflowv2.TrainJob) (client.Object, error) {
	if runtimeJobTemplate == nil || info == nil || trainJob == nil {
		return nil, fmt.Errorf("runtime info or object is missing")
	}

	raw, ok := runtimeJobTemplate.(*jobsetv1alpha2.JobSet)
	if !ok {
		return nil, nil
	}

	var jobSetBuilder *Builder
	oldJobSet := &jobsetv1alpha2.JobSet{}
	if err := j.client.Get(ctx, client.ObjectKeyFromObject(trainJob), oldJobSet); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		jobSetBuilder = NewBuilder(client.ObjectKeyFromObject(trainJob), kubeflowv2.JobSetTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      info.Labels,
				Annotations: info.Annotations,
			},
			Spec: raw.Spec,
		})
		oldJobSet = nil
	} else {
		jobSetBuilder = &Builder{
			JobSet: *oldJobSet.DeepCopy(),
		}
	}

	// TODO (andreyvelich): Add support for the PodSpecOverride.
	// TODO (andreyvelich): Refactor the builder with wrappers for PodSpec.
	jobSet := jobSetBuilder.
		Initializer(trainJob).
		Trainer(info, trainJob).
		PodLabels(info.PodLabels).
		Suspend(trainJob.Spec.Suspend).
		Build()
	if err := ctrlutil.SetControllerReference(trainJob, jobSet, j.scheme); err != nil {
		return nil, err
	}

	if needsCreateOrUpdate(oldJobSet, jobSet, ptr.Deref(trainJob.Spec.Suspend, false)) {
		return jobSet, nil
	}
	return nil, nil
}

func needsCreateOrUpdate(old, new *jobsetv1alpha2.JobSet, trainJobIsSuspended bool) bool {
	return old == nil ||
		(!trainJobIsSuspended && jobSetIsSuspended(old) && !jobSetIsSuspended(new)) ||
		(trainJobIsSuspended && (!equality.Semantic.DeepEqual(old.Spec, new.Spec) || !maps.Equal(old.Labels, new.Labels) || !maps.Equal(old.Annotations, new.Annotations)))
}

func jobSetIsSuspended(jobSet *jobsetv1alpha2.JobSet) bool {
	return ptr.Deref(jobSet.Spec.Suspend, false)
}

func (j *JobSet) TerminalCondition(ctx context.Context, trainJob *kubeflowv2.TrainJob) (*metav1.Condition, error) {
	jobSet := &jobsetv1alpha2.JobSet{}
	if err := j.client.Get(ctx, client.ObjectKeyFromObject(trainJob), jobSet); err != nil {
		return nil, err
	}
	if completed := meta.FindStatusCondition(jobSet.Status.Conditions, string(jobsetv1alpha2.JobSetCompleted)); completed != nil && completed.Status == metav1.ConditionTrue {
		completed.Type = kubeflowv2.TrainJobComplete
		return completed, nil
	}
	if failed := meta.FindStatusCondition(jobSet.Status.Conditions, string(jobsetv1alpha2.JobSetFailed)); failed != nil && failed.Status == metav1.ConditionTrue {
		failed.Type = kubeflowv2.TrainJobFailed
		return failed, nil
	}
	return nil, nil
}

func (j *JobSet) ReconcilerBuilders() []runtime.ReconcilerBuilder {
	if _, err := j.restMapper.RESTMapping(
		schema.GroupKind{Group: jobsetv1alpha2.GroupVersion.Group, Kind: constants.JobSetKind},
		jobsetv1alpha2.SchemeGroupVersion.Version,
	); err != nil {
		// TODO (tenzen-y): After we provide the Configuration API, we should return errors based on the enabled plugins.
		j.logger.Error(err, "JobSet CRDs must be installed in advance")
	}
	return []runtime.ReconcilerBuilder{
		func(b *builder.Builder, c client.Client) *builder.Builder {
			return b.Owns(&jobsetv1alpha2.JobSet{})
		},
	}
}

func (j *JobSet) Validate(runtimeJobTemplate client.Object, runtimeInfo *runtime.Info, oldObj, newObj *kubeflowv2.TrainJob) (admission.Warnings, field.ErrorList) {

	var allErrs field.ErrorList
	specPath := field.NewPath("spec")
	runtimeRefPath := specPath.Child("runtimeRef")

	jobSet, ok := runtimeJobTemplate.(*jobsetv1alpha2.JobSet)
	if !ok {
		return nil, nil
	}

	if newObj.Spec.ModelConfig != nil && newObj.Spec.ModelConfig.Input != nil {
		if !slices.ContainsFunc(jobSet.Spec.ReplicatedJobs, func(x jobsetv1alpha2.ReplicatedJob) bool {
			return x.Name == constants.JobInitializer
		}) {
			allErrs = append(allErrs, field.Invalid(runtimeRefPath, newObj.Spec.RuntimeRef, fmt.Sprintf("trainingRuntime should have %s job when trainJob is configured with input modelConfig", constants.JobInitializer)))
		} else {
			for _, job := range jobSet.Spec.ReplicatedJobs {
				if job.Name == constants.JobInitializer {
					if !slices.ContainsFunc(job.Template.Spec.Template.Spec.InitContainers, func(x corev1.Container) bool {
						return x.Name == constants.ContainerModelInitializer
					}) {
						allErrs = append(allErrs, field.Invalid(runtimeRefPath, newObj.Spec.RuntimeRef, fmt.Sprintf("trainingRuntime should have container with name - %s in the %s job", constants.ContainerModelInitializer, constants.JobInitializer)))
					}
				}
			}
		}
	}

	if newObj.Spec.DatasetConfig != nil {
		if !slices.ContainsFunc(jobSet.Spec.ReplicatedJobs, func(x jobsetv1alpha2.ReplicatedJob) bool {
			return x.Name == constants.JobInitializer
		}) {
			allErrs = append(allErrs, field.Invalid(runtimeRefPath, newObj.Spec.RuntimeRef, fmt.Sprintf("trainingRuntime should have %s job when trainJob is configured with input datasetConfig", constants.JobInitializer)))
		} else {
			for _, job := range jobSet.Spec.ReplicatedJobs {
				if job.Name == constants.JobInitializer {
					if !slices.ContainsFunc(job.Template.Spec.Template.Spec.InitContainers, func(x corev1.Container) bool {
						return x.Name == constants.ContainerDatasetInitializer
					}) {
						allErrs = append(allErrs, field.Invalid(runtimeRefPath, newObj.Spec.RuntimeRef, fmt.Sprintf("trainingRuntime should have container with name - %s in the %s job", constants.ContainerDatasetInitializer, constants.JobInitializer)))
					}
				}
			}
		}
	}

	if len(newObj.Spec.PodSpecOverrides) != 0 {
		podSpecOverridesPath := specPath.Child("podSpecOverrides")
		jobsMap := map[string]bool{}
		for _, job := range jobSet.Spec.ReplicatedJobs {
			jobsMap[job.Name] = true
		}
		// validate if jobOverrides are valid
		for idx, override := range newObj.Spec.PodSpecOverrides {
			for _, job := range override.TargetJobs {
				if _, found := jobsMap[job.Name]; !found {
					allErrs = append(allErrs, field.Invalid(podSpecOverridesPath, newObj.Spec.PodSpecOverrides, fmt.Sprintf("job: %s, configured in the podOverride should be present in the referenced training runtime", job)))
				}
			}
			if len(override.Containers) != 0 {
				// validate if containerOverrides are valid
				containerMap := map[string]bool{}
				for _, job := range jobSet.Spec.ReplicatedJobs {
					for _, container := range job.Template.Spec.Template.Spec.Containers {
						containerMap[container.Name] = true
					}
				}
				containerOverridePath := podSpecOverridesPath.Index(idx)
				for _, container := range override.Containers {
					if _, found := containerMap[container.Name]; !found {
						allErrs = append(allErrs, field.Invalid(containerOverridePath, override.Containers, fmt.Sprintf("container: %s, configured in the containerOverride should be present in the referenced training runtime", container.Name)))
					}
				}
			}
			if len(override.InitContainers) != 0 {
				// validate if initContainerOverrides are valid
				initContainerMap := map[string]bool{}
				for _, job := range jobSet.Spec.ReplicatedJobs {
					for _, initContainer := range job.Template.Spec.Template.Spec.InitContainers {
						initContainerMap[initContainer.Name] = true
					}
				}
				initContainerOverridePath := podSpecOverridesPath.Index(idx)
				for _, container := range override.Containers {
					if _, found := initContainerMap[container.Name]; !found {
						allErrs = append(allErrs, field.Invalid(initContainerOverridePath, override.InitContainers, fmt.Sprintf("initContainer: %s, configured in the initContainerOverride should be present in the referenced training runtime", container.Name)))
					}
				}
			}
		}
	}
	return nil, allErrs
}
