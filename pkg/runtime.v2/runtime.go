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

package runtimev2

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
	kueuelr "sigs.k8s.io/kueue/pkg/util/limitrange"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
)

type Info struct {
	// Labels and Annotations to add to the RuntimeJobTemplate.
	Labels      map[string]string
	Annotations map[string]string
	// Original policy values from the runtime.
	RuntimePolicy RuntimePolicy
	// Trainer parameters to add to the RuntimeJobTemplate.
	Trainer
	// Scheduler parameters to add to the RuntimeJobTemplate.
	*Scheduler
}

type RuntimePolicy struct {
	MLPolicy       *kubeflowv2.MLPolicy
	PodGroupPolicy *kubeflowv2.PodGroupPolicy
}

type Trainer struct {
	NumNodes *int32
	// TODO (andreyvelich). Potentially, we can use map for env and sort it to improve code.
	// Context: https://github.com/kubeflow/training-operator/pull/2308#discussion_r1823267183
	Env           []corev1.EnvVar
	ContainerPort *corev1.ContainerPort
}

// TODO (andreyvelich): Potentially, we can add ScheduleTimeoutSeconds to the Scheduler for consistency.
type Scheduler struct {
	PodLabels     map[string]string
	TotalRequests map[string]TotalResourceRequest
}

type TotalResourceRequest struct {
	Replicas    int32
	PodRequests corev1.ResourceList
}

type InfoOptions struct {
	labels          map[string]string
	annotations     map[string]string
	runtimePolicy   RuntimePolicy
	podSpecReplicas []podSpecReplica
}

type InfoOption func(options *InfoOptions)

var defaultOptions = InfoOptions{}

type podSpecReplica struct {
	replicas int32
	name     string
	podSpec  corev1.PodSpec
}

func WithLabels(labels map[string]string) InfoOption {
	return func(o *InfoOptions) {
		o.labels = maps.Clone(labels)
	}
}

func WithAnnotations(annotations map[string]string) InfoOption {
	return func(o *InfoOptions) {
		o.annotations = maps.Clone(annotations)
	}
}

func WithMLPolicy(mlPolicy *kubeflowv2.MLPolicy) InfoOption {
	return func(o *InfoOptions) {
		o.runtimePolicy.MLPolicy = mlPolicy
	}
}

func WithPodGroupPolicy(pgPolicy *kubeflowv2.PodGroupPolicy) InfoOption {
	return func(o *InfoOptions) {
		o.runtimePolicy.PodGroupPolicy = pgPolicy
	}
}

func WithPodSpecReplicas(replicaName string, replicas int32, podSpec corev1.PodSpec) InfoOption {
	return func(o *InfoOptions) {
		o.podSpecReplicas = append(o.podSpecReplicas, podSpecReplica{
			name:     replicaName,
			replicas: replicas,
			podSpec:  podSpec,
		})
	}
}

func NewInfo(opts ...InfoOption) *Info {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	info := &Info{
		Labels:        make(map[string]string),
		Annotations:   make(map[string]string),
		RuntimePolicy: options.runtimePolicy,
		Scheduler: &Scheduler{
			TotalRequests: make(map[string]TotalResourceRequest, len(options.podSpecReplicas)),
		},
	}

	for _, spec := range options.podSpecReplicas {
		info.TotalRequests[spec.name] = TotalResourceRequest{
			Replicas: spec.replicas,
			// TODO: Need to address LimitRange and RuntimeClass.
			PodRequests: kueuelr.TotalRequests(&spec.podSpec),
		}
	}
	if options.labels != nil {
		info.Labels = options.labels
	}
	if options.annotations != nil {
		info.Annotations = options.annotations
	}

	return info
}
