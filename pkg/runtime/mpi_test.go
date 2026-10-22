/*
Copyright 2026 The Kubeflow Authors.

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

package runtime

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	batchv1ac "k8s.io/client-go/applyconfigurations/batch/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/utils/ptr"
	jobsetv1alpha2ac "sigs.k8s.io/jobset/client-go/applyconfiguration/jobset/v1alpha2"

	trainer "github.com/kubeflow/trainer/v2/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/v2/pkg/constants"
)

func testMPIPolicyWithNodeGroups() *trainer.MPIMLPolicySource {
	return &trainer.MPIMLPolicySource{
		NumProcPerNode: ptr.To[int32](4),
		Launcher:       trainer.MPILauncher{JobName: "launcher"},
		NodeGroups: []trainer.MPINodeGroup{
			{JobName: "cpu", NumNodes: ptr.To[int32](0), NumProcPerNode: ptr.To[int32](32)},
			{JobName: "l40s", ContainerName: "mpi"},
		},
	}
}

func TestHasMPINodeGroups(t *testing.T) {
	cases := map[string]struct {
		mpi  *trainer.MPIMLPolicySource
		want bool
	}{
		"nil MPI policy":            {},
		"MPI policy without groups": {mpi: &trainer.MPIMLPolicySource{NumProcPerNode: ptr.To[int32](1)}},
		"MPI policy with groups":    {mpi: testMPIPolicyWithNodeGroups(), want: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := HasMPINodeGroups(tc.mpi); got != tc.want {
				t.Errorf("HasMPINodeGroups() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMPIGroups(t *testing.T) {
	cases := map[string]struct {
		mpi        *trainer.MPIMLPolicySource
		jobTrainer *trainer.Trainer
		want       []MPIGroup
	}{
		"no groups for an MPI policy without node groups": {
			mpi: &trainer.MPIMLPolicySource{NumProcPerNode: ptr.To[int32](1)},
		},
		"runtime values with defaults": {
			mpi: testMPIPolicyWithNodeGroups(),
			want: []MPIGroup{
				{JobName: "launcher", ContainerName: constants.Node, Launcher: true, NumNodes: 1, NumProcPerNode: 4},
				{JobName: "cpu", ContainerName: constants.Node, Host: true, NumNodes: 0, NumProcPerNode: 32},
				{JobName: "l40s", ContainerName: "mpi", Host: true, NumNodes: 1, NumProcPerNode: 4},
			},
		},
		"TrainJob values replace runtime values field by field and unknown groups are ignored": {
			mpi: testMPIPolicyWithNodeGroups(),
			jobTrainer: &trainer.Trainer{
				Launcher: &trainer.MPILauncherOverride{
					RunAsNode:        ptr.To(true),
					MPIGroupOverride: trainer.MPIGroupOverride{NumProcPerNode: ptr.To[int32](2)},
				},
				NodeGroups: []trainer.MPINodeGroupOverride{
					{Name: "cpu", NumNodes: ptr.To[int32](3)},
					{Name: "l40s", MPIGroupOverride: trainer.MPIGroupOverride{NumProcPerNode: ptr.To[int32](8), Image: ptr.To("image")}},
					{Name: "h100", NumNodes: ptr.To[int32](1)},
				},
			},
			want: []MPIGroup{
				{
					JobName: "launcher", ContainerName: constants.Node, Launcher: true, Host: true, NumNodes: 1, NumProcPerNode: 2,
					NumProcPerNodeFromTrainJob: true,
					Override:                   &trainer.MPIGroupOverride{NumProcPerNode: ptr.To[int32](2)},
				},
				{
					JobName: "cpu", ContainerName: constants.Node, Host: true, NumNodes: 3, NumProcPerNode: 32,
					Override: &trainer.MPIGroupOverride{},
				},
				{
					JobName: "l40s", ContainerName: "mpi", Host: true, NumNodes: 1, NumProcPerNode: 8,
					NumProcPerNodeFromTrainJob: true,
					Override:                   &trainer.MPIGroupOverride{NumProcPerNode: ptr.To[int32](8), Image: ptr.To("image")},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, MPIGroups(tc.mpi, tc.jobTrainer)); len(diff) != 0 {
				t.Errorf("Unexpected groups (-want, +got): %s", diff)
			}
		})
	}
}

func TestFindMPIGroup(t *testing.T) {
	groups := MPIGroups(testMPIPolicyWithNodeGroups(), nil)
	cases := map[string]struct {
		jobName string
		want    *MPIGroup
	}{
		"launcher":      {jobName: "launcher", want: &groups[0]},
		"node group":    {jobName: "l40s", want: &groups[2]},
		"unknown group": {jobName: "h100"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, FindMPIGroup(groups, tc.jobName)); len(diff) != 0 {
				t.Errorf("Unexpected group (-want, +got): %s", diff)
			}
		})
	}
}

func TestExtractContainerResources(t *testing.T) {
	gpuLimits := corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("2")}
	info := NewInfo(WithTemplateSpecObjApply(jobsetv1alpha2ac.JobSetSpec().WithReplicatedJobs(
		jobsetv1alpha2ac.ReplicatedJob().WithName("l40s").WithTemplate(batchv1ac.JobTemplateSpec().
			WithSpec(batchv1ac.JobSpec().WithTemplate(corev1ac.PodTemplateSpec().WithSpec(corev1ac.PodSpec().WithContainers(
				corev1ac.Container().WithName("mpi").WithResources(corev1ac.ResourceRequirements().WithLimits(gpuLimits)),
				corev1ac.Container().WithName("log-shipper"),
			))))),
	)))
	cases := map[string]struct {
		rJobName      string
		containerName string
		want          *corev1.ResourceRequirements
	}{
		"container with resources": {
			rJobName:      "l40s",
			containerName: "mpi",
			want:          &corev1.ResourceRequirements{Limits: gpuLimits, Requests: corev1.ResourceList{}},
		},
		"container without resources": {rJobName: "l40s", containerName: "log-shipper"},
		"unknown replicatedJob":       {rJobName: "h100", containerName: "mpi"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, ExtractContainerResources(info, tc.rJobName, tc.containerName)); len(diff) != 0 {
				t.Errorf("Unexpected resources (-want, +got): %s", diff)
			}
		})
	}
}
