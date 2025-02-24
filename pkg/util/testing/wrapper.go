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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	trainer "github.com/kubeflow/trainer/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/pkg/constants"
	jobsetplugin "github.com/kubeflow/trainer/pkg/runtime/framework/plugins/jobset"
)

type JobSetWrapper struct {
	jobsetv1alpha2.JobSet
}

func MakeJobSetWrapper(namespace, name string) *JobSetWrapper {
	return &JobSetWrapper{
		JobSet: jobsetv1alpha2.JobSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: jobsetv1alpha2.SchemeGroupVersion.String(),
				Kind:       constants.JobSetKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: jobsetv1alpha2.JobSetSpec{
				ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
					{
						Name: constants.JobInitializer,
						Template: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										InitContainers: []corev1.Container{
											{
												Name: constants.ContainerDatasetInitializer,
												VolumeMounts: []corev1.VolumeMount{
													jobsetplugin.VolumeMountDatasetInitializer,
												},
											},
											{
												Name: constants.ContainerModelInitializer,
												VolumeMounts: []corev1.VolumeMount{
													jobsetplugin.VolumeMountModelInitializer,
												},
											},
										},
										Containers: []corev1.Container{
											jobsetplugin.ContainerBusyBox,
										},
										Volumes: []corev1.Volume{
											jobsetplugin.VolumeInitializer,
										},
									},
								},
							},
						},
					},
					{
						Name: constants.JobTrainerNode,
						Template: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: constants.ContainerTrainer,
												VolumeMounts: []corev1.VolumeMount{
													jobsetplugin.VolumeMountDatasetInitializer,
													jobsetplugin.VolumeMountModelInitializer,
												},
											},
										},
										Volumes: []corev1.Volume{
											jobsetplugin.VolumeInitializer,
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

func (j *JobSetWrapper) Replicas(replicas int32) *JobSetWrapper {
	for idx := range j.Spec.ReplicatedJobs {
		j.Spec.ReplicatedJobs[idx].Replicas = replicas
	}
	return j
}

func (j *JobSetWrapper) NumNodes(numNodes int32) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobTrainerNode {
			j.Spec.ReplicatedJobs[i].Template.Spec.Parallelism = &numNodes
			j.Spec.ReplicatedJobs[i].Template.Spec.Completions = &numNodes
		}
	}
	return j
}

func (j *JobSetWrapper) ContainerTrainer(image string, command []string, args []string, res corev1.ResourceList) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobTrainerNode {
			for k, container := range rJob.Template.Spec.Template.Spec.Containers {
				if container.Name == constants.ContainerTrainer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[k].Image = image
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[k].Command = command
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[k].Args = args
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[k].Resources.Requests = res
				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) ContainerTrainerPorts(ports []corev1.ContainerPort) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobTrainerNode {
			for k, container := range rJob.Template.Spec.Template.Spec.Containers {
				if container.Name == constants.ContainerTrainer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[k].Ports = ports
				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) ContainerTrainerEnv(env []corev1.EnvVar) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobTrainerNode {
			for k, container := range rJob.Template.Spec.Template.Spec.Containers {
				if container.Name == constants.ContainerTrainer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[k].Env = env
				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) InitContainerDatasetModelInitializer(image string, command []string, args []string, res corev1.ResourceList) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobInitializer {
			for k, container := range rJob.Template.Spec.Template.Spec.InitContainers {
				if container.Name == constants.ContainerDatasetInitializer || container.Name == constants.ContainerModelInitializer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].Image = image
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].Command = command
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].Args = args
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].Resources.Requests = res
				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) InitContainerDatasetInitializerEnv(env []corev1.EnvVar) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobInitializer {
			for k, container := range rJob.Template.Spec.Template.Spec.InitContainers {
				if container.Name == constants.ContainerDatasetInitializer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].Env = env

				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) InitContainerDatasetInitializerEnvFrom(envFrom []corev1.EnvFromSource) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobInitializer {
			for k, container := range rJob.Template.Spec.Template.Spec.InitContainers {
				if container.Name == constants.ContainerDatasetInitializer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].EnvFrom = envFrom
				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) InitContainerModelInitializerEnv(env []corev1.EnvVar) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobInitializer {
			for k, container := range rJob.Template.Spec.Template.Spec.InitContainers {
				if container.Name == constants.ContainerModelInitializer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].Env = env
				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) InitContainerModelInitializerEnvFrom(envFrom []corev1.EnvFromSource) *JobSetWrapper {
	for i, rJob := range j.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobInitializer {
			for k, container := range rJob.Template.Spec.Template.Spec.InitContainers {
				if container.Name == constants.ContainerModelInitializer {
					j.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[k].EnvFrom = envFrom
				}
			}
		}
	}
	return j
}

func (j *JobSetWrapper) Suspend(suspend bool) *JobSetWrapper {
	j.Spec.Suspend = &suspend
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

func (j *JobSetWrapper) Conditions(conditions ...metav1.Condition) *JobSetWrapper {
	if len(conditions) != 0 {
		j.Status.Conditions = append(j.Status.Conditions, conditions...)
	}
	return j
}

func (j *JobSetWrapper) Obj() *jobsetv1alpha2.JobSet {
	return &j.JobSet
}

type TrainJobWrapper struct {
	trainer.TrainJob
}

func MakeTrainJobWrapper(namespace, name string) *TrainJobWrapper {
	return &TrainJobWrapper{
		TrainJob: trainer.TrainJob{
			TypeMeta: metav1.TypeMeta{
				APIVersion: trainer.SchemeGroupVersion.Version,
				Kind:       trainer.TrainJobKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: trainer.TrainJobSpec{},
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
	runtimeRef := trainer.RuntimeRef{
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

func (t *TrainJobWrapper) Trainer(trainer *trainer.Trainer) *TrainJobWrapper {
	t.Spec.Trainer = trainer
	return t
}

func (t *TrainJobWrapper) DatasetConfig(datasetConfig *trainer.DatasetConfig) *TrainJobWrapper {
	t.Spec.DatasetConfig = datasetConfig
	return t
}

func (t *TrainJobWrapper) ModelConfig(modelConfig *trainer.ModelConfig) *TrainJobWrapper {
	t.Spec.ModelConfig = modelConfig
	return t
}

func (t *TrainJobWrapper) ManagedBy(m string) *TrainJobWrapper {
	t.Spec.ManagedBy = &m
	return t
}

func (t *TrainJobWrapper) Obj() *trainer.TrainJob {
	return &t.TrainJob
}

type TrainJobTrainerWrapper struct {
	trainer.Trainer
}

func MakeTrainJobTrainerWrapper() *TrainJobTrainerWrapper {
	return &TrainJobTrainerWrapper{
		Trainer: trainer.Trainer{},
	}
}

func (t *TrainJobTrainerWrapper) NumNodes(numNodes int32) *TrainJobTrainerWrapper {
	t.Trainer.NumNodes = &numNodes
	return t
}

func (t *TrainJobTrainerWrapper) NumProcPerNode(numProcPerNode intstr.IntOrString) *TrainJobTrainerWrapper {
	t.Trainer.NumProcPerNode = &numProcPerNode
	return t
}

func (t *TrainJobTrainerWrapper) Container(image string, command []string, args []string, resRequests corev1.ResourceList) *TrainJobTrainerWrapper {
	t.Trainer.Image = &image
	t.Trainer.Command = command
	t.Trainer.Args = args
	t.Trainer.ResourcesPerNode = &corev1.ResourceRequirements{
		Requests: resRequests,
	}
	return t
}

func (t *TrainJobTrainerWrapper) ContainerEnv(env []corev1.EnvVar) *TrainJobTrainerWrapper {
	t.Trainer.Env = env
	return t
}

func (t *TrainJobTrainerWrapper) Obj() *trainer.Trainer {
	return &t.Trainer
}

type TrainJobDatasetConfigWrapper struct {
	trainer.DatasetConfig
}

func MakeTrainJobDatasetConfigWrapper() *TrainJobDatasetConfigWrapper {
	return &TrainJobDatasetConfigWrapper{
		DatasetConfig: trainer.DatasetConfig{},
	}
}

func (t *TrainJobDatasetConfigWrapper) StorageUri(storageUri string) *TrainJobDatasetConfigWrapper {
	t.DatasetConfig.StorageUri = &storageUri
	return t
}

func (t *TrainJobDatasetConfigWrapper) ContainerEnv(env []corev1.EnvVar) *TrainJobDatasetConfigWrapper {
	t.DatasetConfig.Env = env
	return t
}

func (t *TrainJobDatasetConfigWrapper) SecretRef(secretRef corev1.LocalObjectReference) *TrainJobDatasetConfigWrapper {
	t.DatasetConfig.SecretRef = &secretRef
	return t
}

func (t *TrainJobDatasetConfigWrapper) Obj() *trainer.DatasetConfig {
	return &t.DatasetConfig
}

type TrainJobModelConfigWrapper struct {
	trainer.ModelConfig
}

func MakeTrainJobModelConfigWrapper() *TrainJobModelConfigWrapper {
	return &TrainJobModelConfigWrapper{
		ModelConfig: trainer.ModelConfig{
			// TODO (andreyvelich): Add support for output model when implemented.
			Input: &trainer.InputModel{},
		},
	}
}

func (t *TrainJobModelConfigWrapper) StorageUri(storageUri string) *TrainJobModelConfigWrapper {
	t.ModelConfig.Input.StorageUri = &storageUri
	return t
}

func (t *TrainJobModelConfigWrapper) ContainerEnv(env []corev1.EnvVar) *TrainJobModelConfigWrapper {
	t.ModelConfig.Input.Env = env
	return t
}

func (t *TrainJobModelConfigWrapper) SecretRef(secretRef corev1.LocalObjectReference) *TrainJobModelConfigWrapper {
	t.ModelConfig.Input.SecretRef = &secretRef
	return t
}

func (t *TrainJobModelConfigWrapper) Obj() *trainer.ModelConfig {
	return &t.ModelConfig
}

type TrainingRuntimeWrapper struct {
	trainer.TrainingRuntime
}

func MakeTrainingRuntimeWrapper(namespace, name string) *TrainingRuntimeWrapper {
	return &TrainingRuntimeWrapper{
		TrainingRuntime: trainer.TrainingRuntime{
			TypeMeta: metav1.TypeMeta{
				APIVersion: trainer.SchemeGroupVersion.String(),
				Kind:       trainer.TrainingRuntimeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: trainer.TrainingRuntimeSpec{
				Template: trainer.JobSetTemplateSpec{
					Spec: jobsetv1alpha2.JobSetSpec{
						ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
							{
								Name: constants.JobInitializer,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												InitContainers: []corev1.Container{
													{
														Name: constants.ContainerDatasetInitializer,
														VolumeMounts: []corev1.VolumeMount{
															jobsetplugin.VolumeMountDatasetInitializer,
														},
													},
													{
														Name: constants.ContainerModelInitializer,
														VolumeMounts: []corev1.VolumeMount{
															jobsetplugin.VolumeMountModelInitializer,
														},
													},
												},
												Containers: []corev1.Container{
													jobsetplugin.ContainerBusyBox,
												},
												Volumes: []corev1.Volume{
													jobsetplugin.VolumeInitializer,
												},
											},
										},
									},
								},
							},
							{
								Name: constants.JobTrainerNode,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{
													{
														Name: constants.ContainerTrainer,
														VolumeMounts: []corev1.VolumeMount{
															jobsetplugin.VolumeMountDatasetInitializer,
															jobsetplugin.VolumeMountModelInitializer,
														},
													},
												},
												Volumes: []corev1.Volume{
													jobsetplugin.VolumeInitializer,
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
		},
	}
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

func (r *TrainingRuntimeWrapper) RuntimeSpec(spec trainer.TrainingRuntimeSpec) *TrainingRuntimeWrapper {
	r.Spec = spec
	return r
}

func (r *TrainingRuntimeWrapper) Obj() *trainer.TrainingRuntime {
	return &r.TrainingRuntime
}

type ClusterTrainingRuntimeWrapper struct {
	trainer.ClusterTrainingRuntime
}

func MakeClusterTrainingRuntimeWrapper(name string) *ClusterTrainingRuntimeWrapper {
	return &ClusterTrainingRuntimeWrapper{
		ClusterTrainingRuntime: trainer.ClusterTrainingRuntime{
			TypeMeta: metav1.TypeMeta{
				APIVersion: trainer.SchemeGroupVersion.String(),
				Kind:       trainer.ClusterTrainingRuntimeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: trainer.TrainingRuntimeSpec{
				Template: trainer.JobSetTemplateSpec{
					Spec: jobsetv1alpha2.JobSetSpec{
						ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
							{
								Name: constants.JobInitializer,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												InitContainers: []corev1.Container{
													{
														Name: constants.ContainerDatasetInitializer,
														VolumeMounts: []corev1.VolumeMount{
															jobsetplugin.VolumeMountDatasetInitializer,
														},
													},
													{
														Name: constants.ContainerModelInitializer,
														VolumeMounts: []corev1.VolumeMount{
															jobsetplugin.VolumeMountModelInitializer,
														},
													},
												},
												Containers: []corev1.Container{
													jobsetplugin.ContainerBusyBox,
												},
												Volumes: []corev1.Volume{
													jobsetplugin.VolumeInitializer,
												},
											},
										},
									},
								},
							},
							{
								Name: constants.JobTrainerNode,
								Template: batchv1.JobTemplateSpec{
									Spec: batchv1.JobSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{
													{
														Name: constants.ContainerTrainer,
														VolumeMounts: []corev1.VolumeMount{
															jobsetplugin.VolumeMountDatasetInitializer,
															jobsetplugin.VolumeMountModelInitializer,
														},
													},
												},
												Volumes: []corev1.Volume{
													jobsetplugin.VolumeInitializer,
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
		},
	}
}

func (r *ClusterTrainingRuntimeWrapper) RuntimeSpec(spec trainer.TrainingRuntimeSpec) *ClusterTrainingRuntimeWrapper {
	r.Spec = spec
	return r
}

func (r *ClusterTrainingRuntimeWrapper) Obj() *trainer.ClusterTrainingRuntime {
	return &r.ClusterTrainingRuntime
}

type TrainingRuntimeSpecWrapper struct {
	trainer.TrainingRuntimeSpec
}

func MakeTrainingRuntimeSpecWrapper(spec trainer.TrainingRuntimeSpec) *TrainingRuntimeSpecWrapper {
	return &TrainingRuntimeSpecWrapper{
		TrainingRuntimeSpec: spec,
	}
}

func (s *TrainingRuntimeSpecWrapper) NumNodes(numNodes int32) *TrainingRuntimeSpecWrapper {
	s.MLPolicy = &trainer.MLPolicy{
		NumNodes: &numNodes,
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) TorchPolicy(numNodes int32, numProcPerNode intstr.IntOrString) *TrainingRuntimeSpecWrapper {
	s.MLPolicy = &trainer.MLPolicy{
		NumNodes: &numNodes,
		MLPolicySource: trainer.MLPolicySource{
			Torch: &trainer.TorchMLPolicySource{
				NumProcPerNode: &numProcPerNode,
			},
		},
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) ContainerTrainer(image string, command []string, args []string, res corev1.ResourceList) *TrainingRuntimeSpecWrapper {
	for i, rJob := range s.Template.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobTrainerNode {
			for j, container := range rJob.Template.Spec.Template.Spec.Containers {
				if container.Name == constants.ContainerTrainer {
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[j].Image = image
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[j].Command = command
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[j].Args = args
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[j].Resources.Requests = res
				}
			}
		}
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) ContainerTrainerEnv(env []corev1.EnvVar) *TrainingRuntimeSpecWrapper {
	for i, rJob := range s.Template.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobTrainerNode {
			for j, container := range rJob.Template.Spec.Template.Spec.Containers {
				if container.Name == constants.ContainerTrainer {
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.Containers[j].Env = env
				}
			}
		}
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) InitContainerDatasetModelInitializer(image string, command []string, args []string, res corev1.ResourceList) *TrainingRuntimeSpecWrapper {
	for i, rJob := range s.Template.Spec.ReplicatedJobs {
		if rJob.Name == constants.JobInitializer {
			for j, container := range rJob.Template.Spec.Template.Spec.InitContainers {
				if container.Name == constants.ContainerDatasetInitializer || container.Name == constants.ContainerModelInitializer {
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[j].Image = image
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[j].Command = command
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[j].Args = args
					s.Template.Spec.ReplicatedJobs[i].Template.Spec.Template.Spec.InitContainers[j].Resources.Requests = res
				}
			}
		}
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) PodGroupPolicyCoscheduling(src *trainer.CoschedulingPodGroupPolicySource) *TrainingRuntimeSpecWrapper {
	s.PodGroupPolicy = &trainer.PodGroupPolicy{
		PodGroupPolicySource: trainer.PodGroupPolicySource{
			Coscheduling: src,
		},
	}
	return s
}

func (s *TrainingRuntimeSpecWrapper) PodGroupPolicyCoschedulingSchedulingTimeout(timeout int32) *TrainingRuntimeSpecWrapper {
	if s.PodGroupPolicy == nil || s.PodGroupPolicy.Coscheduling == nil {
		return s.PodGroupPolicyCoscheduling(&trainer.CoschedulingPodGroupPolicySource{
			ScheduleTimeoutSeconds: &timeout,
		})
	}
	s.PodGroupPolicy.Coscheduling.ScheduleTimeoutSeconds = &timeout
	return s
}

func (s *TrainingRuntimeSpecWrapper) Obj() trainer.TrainingRuntimeSpec {
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
				Kind:       constants.PodGroupKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
		},
	}
}

func (p *SchedulerPluginsPodGroupWrapper) MinMember(members int32) *SchedulerPluginsPodGroupWrapper {
	p.PodGroup.Spec.MinMember = members
	return p
}

func (p *SchedulerPluginsPodGroupWrapper) MinResources(resources corev1.ResourceList) *SchedulerPluginsPodGroupWrapper {
	p.PodGroup.Spec.MinResources = resources
	return p
}

func (p *SchedulerPluginsPodGroupWrapper) SchedulingTimeout(timeout int32) *SchedulerPluginsPodGroupWrapper {
	p.PodGroup.Spec.ScheduleTimeoutSeconds = &timeout
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
