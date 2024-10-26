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
	Policy
	Labels      map[string]string
	PodLabels   map[string]string
	Annotations map[string]string
	Trainer
	TotalRequests map[string]TotalResourceRequest
}

type Trainer struct {
	// TODO (andreyvelich): Add more parameters.
	NumNodes *int32
	Env      map[string]string
}

type Policy struct {
	MLPolicy       *kubeflowv2.MLPolicy
	PodGroupPolicy *kubeflowv2.PodGroupPolicy
}

type TotalResourceRequest struct {
	Replicas    int32
	PodRequests corev1.ResourceList
}

type InfoOptions struct {
	podSpecReplicas []podSpecReplica
	Policy
	labels      map[string]string
	annotations map[string]string
}

type InfoOption func(options *InfoOptions)

var defaultOptions = InfoOptions{}

type podSpecReplica struct {
	replicas int32
	name     string
	podSpec  corev1.PodSpec
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

func WithPodGroupPolicy(pgPolicy *kubeflowv2.PodGroupPolicy) InfoOption {
	return func(o *InfoOptions) {
		o.PodGroupPolicy = pgPolicy
	}
}

func WithMLPolicy(mlPolicy *kubeflowv2.MLPolicy) InfoOption {
	return func(o *InfoOptions) {
		o.MLPolicy = mlPolicy
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
		TotalRequests: make(map[string]TotalResourceRequest, len(options.podSpecReplicas)),
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
	info.Policy = options.Policy
	return info
}
