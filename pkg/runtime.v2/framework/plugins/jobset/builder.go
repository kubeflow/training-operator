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
	"maps"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
)

type Builder struct {
	*jobsetv1alpha2.JobSet
}

func NewBuilder(objectKey client.ObjectKey, jobSetTemplateSpec kubeflowv2.JobSetTemplateSpec) *Builder {
	return &Builder{
		JobSet: &jobsetv1alpha2.JobSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: jobsetv1alpha2.SchemeGroupVersion.String(),
				Kind:       "JobSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   objectKey.Namespace,
				Name:        objectKey.Name,
				Labels:      maps.Clone(jobSetTemplateSpec.Labels),
				Annotations: maps.Clone(jobSetTemplateSpec.Annotations),
			},
			Spec: *jobSetTemplateSpec.Spec.DeepCopy(),
		},
	}
}

func (b *Builder) ContainerImage(image *string) *Builder {
	if image == nil || *image == "" {
		return b
	}
	for i, rJob := range b.Spec.ReplicatedJobs {
		for j := range rJob.Template.Spec.Template.Spec.Containers {
			b.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[j].Image = *image
		}
	}
	return b
}

func (b *Builder) JobCompletionMode(mode batchv1.CompletionMode) *Builder {
	for i := range b.Spec.ReplicatedJobs {
		b.Spec.ReplicatedJobs[i].Template.Spec.CompletionMode = &mode
	}
	return b
}

// TODO: Supporting merge labels would be great.
func (b *Builder) PodLabels(labels map[string]string) *Builder {
	for i := range b.Spec.ReplicatedJobs {
		b.Spec.ReplicatedJobs[i].Template.Spec.Template.Labels = labels
	}
	return b
}

// TODO: Need to support all TrainJob fields.

func (b *Builder) Build() *jobsetv1alpha2.JobSet {
	return b.JobSet
}
