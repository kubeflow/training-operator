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

package testing

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
)

type JobSetWrapper struct {
	jobsetv1alpha2.JobSet
}

func MakeJobSetWrapper(namespace, name string) *JobSetWrapper {
	return &JobSetWrapper{
		JobSet: jobsetv1alpha2.JobSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: jobsetv1alpha2.SchemeGroupVersion.String(),
				Kind:       "JobSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: jobsetv1alpha2.JobSetSpec{
				ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
					{
						Name:     "Coordinator",
						Replicas: 1,
						Template: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: "trainer",
											},
										},
									},
								},
							},
						},
					},
					{
						Name:     "Worker",
						Replicas: 1,
						Template: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: "trainer",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (j *JobSetWrapper) Suspend(suspend bool) *JobSetWrapper {
	j.Spec.Suspend = &suspend
	return j
}

func (j *JobSetWrapper) Completions(idx int, completions int32) *JobSetWrapper {
	if len(j.Spec.ReplicatedJobs) < idx {
		return j
	}
	j.Spec.ReplicatedJobs[idx].Template.Spec.Completions = &completions
	return j
}

func (j *JobSetWrapper) Parallelism(idx int, parallelism int32) *JobSetWrapper {
	if len(j.Spec.ReplicatedJobs) < idx {
		return j
	}
	j.Spec.ReplicatedJobs[idx].Template.Spec.Parallelism = &parallelism
	return j
}

func (j *JobSetWrapper) ResourceRequests(idx int, res corev1.ResourceList) *JobSetWrapper {
	if len(j.Spec.ReplicatedJobs) < idx {
		return j
	}
	j.Spec.ReplicatedJobs[idx].Template.Spec.Template.Spec.Containers[0].Resources.Requests = res
	return j
}

func (j *JobSetWrapper) JobCompletionMode(mode batchv1.CompletionMode) *JobSetWrapper {
	for i := range j.Spec.ReplicatedJobs {
		j.Spec.ReplicatedJobs[i].Template.Spec.CompletionMode = &mode
	}
	return j
}

func (j *JobSetWrapper) ContainerImage(image *string) *JobSetWrapper {
	if image == nil || *image == "" {
		return j
	}
	for i, rJob := range j.Spec.ReplicatedJobs {
		for k := range rJob.Template.Spec.Template.Spec.Containers {
			j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[k].Image = *image
		}
	}
	return j
}

func (j *JobSetWrapper) ControllerReference(gvk schema.GroupVersionKind, name, uid string) *JobSetWrapper {
	j.OwnerReferences = append(j.OwnerReferences, metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               name,
		UID:                types.UID(uid),
		Controller:         ptr.To(true),
		BlockOwnerDeletion: ptr.To(true),
	})
	return j
}

func (j *JobSetWrapper) PodLabel(key, value string) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Template.Spec.Template.Labels == nil {
			j.Spec.ReplicatedJobs[i].Template.Spec.Template.Labels = make(map[string]string, 1)
		}
		j.Spec.ReplicatedJobs[i].Template.Spec.Template.Labels[key] = value
	}
	return j
}

func (j *JobSetWrapper) Label(key, value string) *JobSetWrapper {
	if j.ObjectMeta.Labels == nil {
		j.ObjectMeta.Labels = make(map[string]string, 1)
	}
	j.ObjectMeta.Labels[key] = value
	return j
}

func (j *JobSetWrapper) Annotation(key, value string) *JobSetWrapper {
	if j.ObjectMeta.Annotations == nil {
		j.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	j.ObjectMeta.Annotations[key] = value
	return j
}

func (j *JobSetWrapper) Replicas(replicas int32) *JobSetWrapper {
	for idx := range j.Spec.ReplicatedJobs {
		j.Spec.ReplicatedJobs[idx].Replicas = replicas
	}
	return j
}

func (j *JobSetWrapper) Clone() *JobSetWrapper {
	return &JobSetWrapper{
		JobSet: *j.JobSet.DeepCopy(),
	}
}

func (j *JobSetWrapper) Obj() *jobsetv1alpha2.JobSet {
	return &j.JobSet
}

type TrainJobWrapper struct {
	kubeflowv2.TrainJob
}

func MakeTrainJobWrapper(namespace, name string) *TrainJobWrapper {
	return &TrainJobWrapper{
		TrainJob: kubeflowv2.TrainJob{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kubeflowv2.SchemeGroupVersion.Version,
				Kind:       "TrainJob",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: kubeflowv2.TrainJobSpec{},
		},
	}
}

func (t *TrainJobWrapper) Suspend(suspend bool) *TrainJobWrapper {
	t.Spec.Suspend = &suspend
	return t
}

func (t *TrainJobWrapper) UID(uid string) *TrainJobWrapper {
	t.ObjectMeta.UID = types.UID(uid)
	return t
}

func (t *TrainJobWrapper) SpecLabel(key, value string) *TrainJobWrapper {
	if t.Spec.Labels == nil {
		t.Spec.Labels = make(map[string]string, 1)
	}
	t.Spec.Labels[key] = value
	return t
}

func (t *TrainJobWrapper) SpecAnnotation(key, value string) *TrainJobWrapper {
	if t.Spec.Annotations == nil {
		t.Spec.Annotations = make(map[string]string, 1)
	}
	t.Spec.Annotations[key] = value
	return t
}

func (t *TrainJobWrapper) RuntimeRef(gvk schema.GroupVersionKind, name string) *TrainJobWrapper {
	runtimeRef := kubeflowv2.RuntimeRef{
		Name: name,
	}
	if gvk.Group != "" {
		runtimeRef.APIGroup = &gvk.Group
	}
	if gvk.Kind != "" {
		runtimeRef.Kind = &gvk.Kind
	}
	t.Spec.RuntimeRef = runtimeRef
	return t
}

func (t *TrainJobWrapper) Trainer(trainer *kubeflowv2.Trainer) *TrainJobWrapper {
	t.Spec.Trainer = trainer
	return t
}

func (t *TrainJobWrapper) ManagedBy(m string) *TrainJobWrapper {
	t.Spec.ManagedBy = &m
	return t
}
func (t *TrainJobWrapper) ModelConfig(config *kubeflowv2.ModelConfig) *TrainJobWrapper {
	t.Spec.ModelConfig = config
	return t
}

func (t *TrainJobWrapper) Obj() *kubeflowv2.TrainJob {
	return &t.TrainJob
}

type TrainJobTrainerWrapper struct {
	kubeflowv2.Trainer
}

func MakeTrainJobTrainerWrapper() *TrainJobTrainerWrapper {
	return &TrainJobTrainerWrapper{
		Trainer: kubeflowv2.Trainer{},
	}
}

func (t *TrainJobTrainerWrapper) ContainerImage(img string) *TrainJobTrainerWrapper {
	t.Image = &img
	return t
}

func (t *TrainJobTrainerWrapper) Obj() *kubeflowv2.Trainer {
	return &t.Trainer
}

type TrainingRuntimeWrapper struct {
	kubeflowv2.TrainingRuntime
}

func MakeTrainingRuntimeWrapper(namespace, name string) *TrainingRuntimeWrapper {
	return &TrainingRuntimeWrapper{
		TrainingRuntime: kubeflowv2.TrainingRuntime{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kubeflowv2.SchemeGroupVersion.String(),
				Kind:       kubeflowv2.TrainingRuntimeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: kubeflowv2.TrainingRuntimeSpec{
				Template: kubeflowv2.JobSetTemplateSpec{
					Spec: jobsetv1alpha2.JobSetSpec{
						ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
							{
								Name:     "Coordinator",
								Replicas: 1,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{{
													Name: "trainer",
												}},
											},
										},
									},
								},
							},
							{
								Name:     "Worker",
								Replicas: 1,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{{
													Name: "trainer",
												}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *TrainingRuntimeWrapper) RuntimeSpec(spec kubeflowv2.TrainingRuntimeSpec) *TrainingRuntimeWrapper {
	r.Spec = spec
	return r
}

func (r *TrainingRuntimeWrapper) Label(key, value string) *TrainingRuntimeWrapper {
	if r.ObjectMeta.Labels == nil {
		r.ObjectMeta.Labels = make(map[string]string, 1)
	}
	r.ObjectMeta.Labels[key] = value
	return r
}

func (r *TrainingRuntimeWrapper) Annotation(key, value string) *TrainingRuntimeWrapper {
	if r.ObjectMeta.Annotations == nil {
		r.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	r.ObjectMeta.Annotations[key] = value
	return r
}

func (r *TrainingRuntimeWrapper) Clone() *TrainingRuntimeWrapper {
	return &TrainingRuntimeWrapper{
		TrainingRuntime: *r.TrainingRuntime.DeepCopy(),
	}
}

func (r *TrainingRuntimeWrapper) Obj() *kubeflowv2.TrainingRuntime {
	return &r.TrainingRuntime
}

type ClusterTrainingRuntimeWrapper struct {
	kubeflowv2.ClusterTrainingRuntime
}

func MakeClusterTrainingRuntimeWrapper(name string) *ClusterTrainingRuntimeWrapper {
	return &ClusterTrainingRuntimeWrapper{
		ClusterTrainingRuntime: kubeflowv2.ClusterTrainingRuntime{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kubeflowv2.SchemeGroupVersion.String(),
				Kind:       kubeflowv2.ClusterTrainingRuntimeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: kubeflowv2.TrainingRuntimeSpec{
				Template: kubeflowv2.JobSetTemplateSpec{
					Spec: jobsetv1alpha2.JobSetSpec{
						ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
							{
								Name:     "Coordinator",
								Replicas: 1,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{{
													Name: "trainer",
												}},
											},
										},
									},
								},
							},
							{
								Name:     "Worker",
								Replicas: 1,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{{
													Name: "trainer",
												}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ClusterTrainingRuntimeWrapper) RuntimeSpec(spec kubeflowv2.TrainingRuntimeSpec) *ClusterTrainingRuntimeWrapper {
	r.Spec = spec
	return r
}

func (r *ClusterTrainingRuntimeWrapper) Clone() *ClusterTrainingRuntimeWrapper {
	return &ClusterTrainingRuntimeWrapper{
		ClusterTrainingRuntime: *r.ClusterTrainingRuntime.DeepCopy(),
	}
}

func (r *ClusterTrainingRuntimeWrapper) Obj() *kubeflowv2.ClusterTrainingRuntime {
	return &r.ClusterTrainingRuntime
}

type TrainingRuntimeSpecWrapper struct {
	kubeflowv2.TrainingRuntimeSpec
}

func MakeTrainingRuntimeSpecWrapper(spec kubeflowv2.TrainingRuntimeSpec) *TrainingRuntimeSpecWrapper {
	return &TrainingRuntimeSpecWrapper{
		TrainingRuntimeSpec: spec,
	}
}

func (s *TrainingRuntimeSpecWrapper) Replicas(replicas int32) *TrainingRuntimeSpecWrapper {
	for idx := range s.Template.Spec.ReplicatedJobs {
		s.Template.Spec.ReplicatedJobs[idx].Replicas = replicas
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) ContainerImage(image string) *TrainingRuntimeSpecWrapper {
	for i, rJob := range s.Template.Spec.ReplicatedJobs {
		for j := range rJob.Template.Spec.Template.Spec.Containers {
			s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[j].Image = image
		}
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) ResourceRequests(idx int, res corev1.ResourceList) *TrainingRuntimeSpecWrapper {
	if len(s.Template.Spec.ReplicatedJobs) < idx {
		return s
	}
	s.Template.Spec.ReplicatedJobs[idx].Template.Spec.Template.Spec.Containers[0].Resources.Requests = res
	return s
}

func (s *TrainingRuntimeSpecWrapper) PodGroupPolicyCoscheduling(src *kubeflowv2.CoschedulingPodGroupPolicySource) *TrainingRuntimeSpecWrapper {
	if s.PodGroupPolicy == nil {
		s.PodGroupPolicy = &kubeflowv2.PodGroupPolicy{}
	}
	s.PodGroupPolicy.Coscheduling = src
	return s
}

func (s *TrainingRuntimeSpecWrapper) PodGroupPolicyCoschedulingSchedulingTimeout(timeout int32) *TrainingRuntimeSpecWrapper {
	if s.PodGroupPolicy == nil || s.PodGroupPolicy.Coscheduling == nil {
		return s.PodGroupPolicyCoscheduling(&kubeflowv2.CoschedulingPodGroupPolicySource{
			ScheduleTimeoutSeconds: &timeout,
		})
	}
	s.PodGroupPolicy.Coscheduling.ScheduleTimeoutSeconds = &timeout
	return s
}

func (s *TrainingRuntimeSpecWrapper) MLPolicyNumNodes(numNodes int32) *TrainingRuntimeSpecWrapper {
	if s.MLPolicy == nil {
		s.MLPolicy = &kubeflowv2.MLPolicy{}
	}
	s.MLPolicy.NumNodes = &numNodes
	return s
}

func (s *TrainingRuntimeSpecWrapper) Obj() kubeflowv2.TrainingRuntimeSpec {
	return s.TrainingRuntimeSpec
}

type SchedulerPluginsPodGroupWrapper struct {
	schedulerpluginsv1alpha1.PodGroup
}

func MakeSchedulerPluginsPodGroup(namespace, name string) *SchedulerPluginsPodGroupWrapper {
	return &SchedulerPluginsPodGroupWrapper{
		PodGroup: schedulerpluginsv1alpha1.PodGroup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: schedulerpluginsv1alpha1.SchemeGroupVersion.String(),
				Kind:       "PodGroup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
		},
	}
}

func (p *SchedulerPluginsPodGroupWrapper) SchedulingTimeout(timeout int32) *SchedulerPluginsPodGroupWrapper {
	p.PodGroup.Spec.ScheduleTimeoutSeconds = &timeout
	return p
}

func (p *SchedulerPluginsPodGroupWrapper) MinMember(members int32) *SchedulerPluginsPodGroupWrapper {
	p.PodGroup.Spec.MinMember = members
	return p
}

func (p *SchedulerPluginsPodGroupWrapper) MinResources(resources corev1.ResourceList) *SchedulerPluginsPodGroupWrapper {
	p.PodGroup.Spec.MinResources = resources
	return p
}

func (p *SchedulerPluginsPodGroupWrapper) ControllerReference(gvk schema.GroupVersionKind, name, uid string) *SchedulerPluginsPodGroupWrapper {
	p.OwnerReferences = append(p.OwnerReferences, metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               name,
		UID:                types.UID(uid),
		Controller:         ptr.To(true),
		BlockOwnerDeletion: ptr.To(true),
	})
	return p
}

func (p *SchedulerPluginsPodGroupWrapper) Obj() *schedulerpluginsv1alpha1.PodGroup {
	return &p.PodGroup
}
