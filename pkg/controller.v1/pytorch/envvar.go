// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package pytorch

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

const (
	// Worker/node size related arguments.

	// EnvNprocPerNode is the environment variable name for the number of processes per node.
	EnvNprocPerNode = "PET_NPROC_PER_NODE"
	// EnvNnodes is the environment variable name for the number of nodes.
	EnvNnodes = "PET_NNODES"
	// EnvNodeRank is the environment variable name for the rank of nodes.
	EnvNodeRank = "PET_NODE_RANK"
)

// EnvVarGenerator is the environment variable generator interface.
type EnvVarGenerator interface {
	Generate(job *kubeflowv1.PyTorchJob) ([]corev1.EnvVar, error)
}

func setPodEnv(obj interface{}, podTemplateSpec *corev1.PodTemplateSpec, rtype, index string) error {
	pytorchjob, ok := obj.(*kubeflowv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", obj)
	}

	for i := range podTemplateSpec.Spec.Containers {
		// Initialize the environment variables.
		if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
			podTemplateSpec.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
		}
		// Set PYTHONUNBUFFERED to true, to disable output buffering.
		// Ref https://stackoverflow.com/questions/59812009/what-is-the-use-of-pythonunbuffered-in-docker-file.
		podTemplateSpec.Spec.Containers[i].Env = append(
			podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "PYTHONUNBUFFERED",
				Value: "0",
			})

		totalReplicas := getTotalReplicas(pytorchjob)
		nprocPerNode := getNprocPerNodeInt(pytorchjob)
		worldSize := int(totalReplicas) * nprocPerNode

		// If the master is not null, then we need to set the MASTER_ADDR and RANK.
		if pytorchjob.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeMaster] != nil {
			envVars, err := GetMasterEnvVarGenerator().Generate(pytorchjob)
			if err != nil {
				return err
			}
			// Set master related environment variables.
			podTemplateSpec.Spec.Containers[i].Env = append(
				podTemplateSpec.Spec.Containers[i].Env, envVars...)

			// Set world size and rank.
			rank, err := strconv.Atoi(index)
			if err != nil {
				return err
			}
			if rtype == strings.ToLower(string(kubeflowv1.PyTorchJobReplicaTypeWorker)) {
				rank = rank + 1
			}

			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "WORLD_SIZE",
				Value: strconv.Itoa(worldSize),
			})
			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "RANK",
				Value: strconv.Itoa(rank),
			})
			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  EnvNprocPerNode,
				Value: *pytorchjob.Spec.NprocPerNode,
			})
			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  EnvNodeRank,
				Value: strconv.Itoa(rank),
			})
		}

		// Set the elastic environment variables if the elasticPolicy is not null.
		// nnodes is set in range format in elastic mode, e.g. nnodes=1:4
		// otherwise, nnodes is set by int, e.g. nnodes=2
		if pytorchjob.Spec.ElasticPolicy != nil {
			envVars, err := GetElasticEnvVarGenerator().Generate(pytorchjob)
			if err != nil {
				return err
			}
			// Set elastic related environment variables.
			podTemplateSpec.Spec.Containers[i].Env = append(
				podTemplateSpec.Spec.Containers[i].Env, envVars...)
		} else {
			podTemplateSpec.Spec.Containers[i].Env = append(
				podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  EnvNnodes,
					Value: strconv.Itoa(int(totalReplicas)),
				})
		}
	}

	return nil
}

// getNprocPerNodeInt return the int value of NprocPerNode, return 1 if not int
// When nproc_per_node set to auto, it means the number of process will be determinated
// in the user process phase, in this case, world size env will not be used.
func getNprocPerNodeInt(job *kubeflowv1.PyTorchJob) int {
	if job.Spec.NprocPerNode == nil {
		return 1
	}
	if np, err := strconv.Atoi(*job.Spec.NprocPerNode); err == nil {
		return np
	}
	return 1
}

func getTotalReplicas(job *kubeflowv1.PyTorchJob) int32 {
	jobReplicas := int32(0)
	for _, r := range job.Spec.PyTorchReplicaSpecs {
		jobReplicas += *r.Replicas
	}
	return jobReplicas
}

func replicaName(jobName string, rtype kubeflowv1.ReplicaType, index int) string {
	n := jobName + "-" + strings.ToLower(string(rtype)) + "-" + strconv.Itoa(index)
	return strings.Replace(n, "/", "-", -1)
}

func getPortFromPyTorchJob(job *kubeflowv1.PyTorchJob, rtype kubeflowv1.ReplicaType) (int32, error) {
	containers := job.Spec.PyTorchReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == kubeflowv1.PytorchJobDefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == kubeflowv1.PytorchJobDefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("port not found")
}
