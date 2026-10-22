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
	"cmp"

	"k8s.io/utils/ptr"

	trainer "github.com/kubeflow/trainer/v2/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/v2/pkg/constants"
)

// MPIGroup is the launcher or a node group of an MPI runtime that declares node groups,
// with the TrainJob overrides applied.
type MPIGroup struct {
	// JobName is the name of the replicatedJob of the group.
	JobName string
	// ContainerName is the name of the MPI container in the Pods of the group.
	ContainerName string
	// Launcher is true for the launcher.
	Launcher bool
	// Host is true when the Pods of the group get lines in the hostfile: always for a node group,
	// and for the launcher when it runs as a node.
	Host bool
	// NumNodes is the number of Pods of the group. It is always 1 for the launcher.
	NumNodes int32
	// NumProcPerNode is the number of slots of each host of the group, before the GPU fallback.
	NumProcPerNode int32
	// NumProcPerNodeFromTrainJob is true when the TrainJob sets numProcPerNode for the group.
	NumProcPerNodeFromTrainJob bool
	// Override is the TrainJob override of the group, or nil when the TrainJob does not override it.
	Override *trainer.MPIGroupOverride
}

// HasMPINodeGroups returns true when the MPI policy declares a launcher and node groups.
func HasMPINodeGroups(mpi *trainer.MPIMLPolicySource) bool {
	return mpi != nil && mpi.Launcher.JobName != "" && len(mpi.NodeGroups) > 0
}

// MPIGroups returns the launcher followed by the node groups, in the runtime order, of an MPI
// policy that declares node groups. TrainJob values replace runtime values field by field, and
// mlPolicy.mpi.numProcPerNode is the default number of slots. TrainJob entries that do not
// match a node group of the runtime are ignored, since the TrainJob webhook rejects them.
func MPIGroups(mpi *trainer.MPIMLPolicySource, jobTrainer *trainer.Trainer) []MPIGroup {
	if !HasMPINodeGroups(mpi) {
		return nil
	}
	defaultSlots := ptr.Deref(mpi.NumProcPerNode, 1)
	launcher := MPIGroup{
		JobName:        mpi.Launcher.JobName,
		ContainerName:  cmp.Or(mpi.Launcher.ContainerName, constants.Node),
		Launcher:       true,
		Host:           ptr.Deref(mpi.Launcher.RunAsNode, false),
		NumNodes:       1,
		NumProcPerNode: ptr.Deref(mpi.Launcher.NumProcPerNode, defaultSlots),
	}
	overrides := make(map[string]*trainer.MPINodeGroupOverride)
	if jobTrainer != nil {
		if o := jobTrainer.Launcher; o != nil {
			if o.RunAsNode != nil {
				launcher.Host = *o.RunAsNode
			}
			if o.NumProcPerNode != nil {
				launcher.NumProcPerNode = *o.NumProcPerNode
				launcher.NumProcPerNodeFromTrainJob = true
			}
			launcher.Override = &o.MPIGroupOverride
		}
		for i := range jobTrainer.NodeGroups {
			overrides[jobTrainer.NodeGroups[i].Name] = &jobTrainer.NodeGroups[i]
		}
	}
	groups := []MPIGroup{launcher}
	for _, nodeGroup := range mpi.NodeGroups {
		group := MPIGroup{
			JobName:        nodeGroup.JobName,
			ContainerName:  cmp.Or(nodeGroup.ContainerName, constants.Node),
			Host:           true,
			NumNodes:       ptr.Deref(nodeGroup.NumNodes, 1),
			NumProcPerNode: ptr.Deref(nodeGroup.NumProcPerNode, defaultSlots),
		}
		if o, ok := overrides[nodeGroup.JobName]; ok {
			if o.NumNodes != nil {
				group.NumNodes = *o.NumNodes
			}
			if o.NumProcPerNode != nil {
				group.NumProcPerNode = *o.NumProcPerNode
				group.NumProcPerNodeFromTrainJob = true
			}
			group.Override = &o.MPIGroupOverride
		}
		groups = append(groups, group)
	}
	return groups
}

// FindMPIGroup returns the group of the given replicatedJob, or nil when it is not part of the groups.
func FindMPIGroup(groups []MPIGroup, jobName string) *MPIGroup {
	for i := range groups {
		if groups[i].JobName == jobName {
			return &groups[i]
		}
	}
	return nil
}
