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
	"k8s.io/apimachinery/pkg/util/validation/field"
	"maps"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
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
var _ framework.CustomValidationPlugin = (*JobSet)(nil)

const Name = "JobSet"

//+kubebuilder:rbac:groups=jobset.x-k8s.io,resources=jobsets,verbs=get;list;watch;create

func New(ctx context.Context, c client.Client, _ client.FieldIndexer) (framework.Plugin, error) {
	return &JobSet{
		client:     c,
		restMapper: c.RESTMapper(),
		scheme:     c.Scheme(),
		logger:     ctrl.LoggerFrom(ctx).WithValues("pluginName", "JobSet"),
	}, nil
}

func (j *JobSet) Name() string {
	return Name
}

func (j *JobSet) Build(ctx context.Context, info *runtime.Info, trainJob *kubeflowv2.TrainJob) (client.Object, error) {
	if info == nil || info.Obj == nil || trainJob == nil {
		return nil, fmt.Errorf("runtime info or object is missing")
	}
	raw, ok := info.Obj.(*jobsetv1alpha2.JobSet)
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

	// TODO (tenzen-y): We should support all field propagation in builder.
	jobSet := jobSetBuilder.
		Suspend(trainJob.Spec.Suspend).
		ContainerImage(trainJob.Spec.Trainer.Image).
		JobCompletionMode(batchv1.IndexedCompletion).
		PodLabels(info.PodLabels).
		Build()
	if err := ctrlutil.SetControllerReference(trainJob, jobSet, j.scheme); err != nil {
		return nil, err
	}
	if err := info.Update(jobSet); err != nil {
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

func (j *JobSet) ReconcilerBuilders() []runtime.ReconcilerBuilder {
	if _, err := j.restMapper.RESTMapping(
		schema.GroupKind{Group: jobsetv1alpha2.GroupVersion.Group, Kind: "JobSet"},
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

func (j *JobSet) Validate(oldObj, newObj *kubeflowv2.TrainJob, runtimeInfo *runtime.Info) (admission.Warnings, field.ErrorList) {

	var allErrs field.ErrorList
	specPath := field.NewPath("spec")

	jobSet, ok := runtimeInfo.Obj.(*jobsetv1alpha2.JobSet)
	if !ok {
		return nil, nil
	}

	if newObj.Spec.ModelConfig != nil {
		// validate `model-initializer` container in the `Initializer` Job
		if newObj.Spec.ModelConfig.Input != nil {
			modelConfigInputPath := specPath.Child("modelConfig").Child("input")
			if len(jobSet.Spec.ReplicatedJobs) == 0 {
				allErrs = append(allErrs, field.Invalid(modelConfigInputPath, newObj.Spec.ModelConfig.Input, "trainingRuntime should have replicated jobs configured with model config input set"))
			} else {
				initializerJobFound := false
				modelInitializerContainerFound := false
				for _, job := range jobSet.Spec.ReplicatedJobs {
					if job.Name == "Initializer" {
						initializerJobFound = true
						for _, container := range job.Template.Spec.Template.Spec.Containers {
							if container.Name == "model-initializer" {
								modelInitializerContainerFound = true
							}
						}
					}
				}
				if !initializerJobFound {
					allErrs = append(allErrs, field.Invalid(modelConfigInputPath, newObj.Spec.ModelConfig.Input, "trainingRuntime should have replicated job configured with name - Initializer"))
				} else if !modelInitializerContainerFound {
					allErrs = append(allErrs, field.Invalid(modelConfigInputPath, newObj.Spec.ModelConfig.Input, "trainingRuntime with replicated job initializer should have container with name - model-initializer"))
				}
			}
		}

		// validate `model-exporter` container in the `Exporter` Job
		if newObj.Spec.ModelConfig.Output != nil {
			modelConfigInputPath := specPath.Child("modelConfig").Child("output")
			if len(jobSet.Spec.ReplicatedJobs) == 0 {
				allErrs = append(allErrs, field.Invalid(modelConfigInputPath, newObj.Spec.ModelConfig.Output, "trainingRuntime should have replicated jobs configured with model config output set"))
			} else {
				exporterJobFound := false
				modelExporterContainerFound := false
				for _, job := range jobSet.Spec.ReplicatedJobs {
					if job.Name == "Exporter" {
						exporterJobFound = true
						for _, container := range job.Template.Spec.Template.Spec.Containers {
							if container.Name == "model-exporter" {
								modelExporterContainerFound = true
							}
						}
					}
				}
				if !exporterJobFound {
					allErrs = append(allErrs, field.Invalid(modelConfigInputPath, newObj.Spec.ModelConfig.Input, "trainingRuntime should have replicated job configured with name - Exporter"))
				} else if !modelExporterContainerFound {
					allErrs = append(allErrs, field.Invalid(modelConfigInputPath, newObj.Spec.ModelConfig.Input, "trainingRuntime with replicated job initializer should have contianer with name - model-exporter"))
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
				if ok, _ := jobsMap[job.Name]; !ok {
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
					if _, ok := containerMap[container.Name]; !ok {
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
					if _, ok := initContainerMap[container.Name]; !ok {
						allErrs = append(allErrs, field.Invalid(initContainerOverridePath, override.InitContainers, fmt.Sprintf("initContainer: %s, configured in the initContainerOverride should be present in the referenced training runtime", container.Name)))
					}
				}
			}
		}
	}
	return nil, allErrs
}
