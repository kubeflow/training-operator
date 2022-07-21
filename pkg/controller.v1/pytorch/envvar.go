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

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
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

			totalReplicas := getTotalReplicas(pytorchjob)

			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "WORLD_SIZE",
				Value: strconv.Itoa(int(totalReplicas)),
			})
			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "RANK",
				Value: strconv.Itoa(rank),
			})
		}

		// Set the elastic environment variables if the elasticPolicy is not null.
		if pytorchjob.Spec.ElasticPolicy != nil {
			envVars, err := GetElasticEnvVarGenerator().Generate(pytorchjob)
			if err != nil {
				return err
			}
			// Set elastic related environment variables.
			podTemplateSpec.Spec.Containers[i].Env = append(
				podTemplateSpec.Spec.Containers[i].Env, envVars...)
		}
	}

	return nil
}

func getTotalReplicas(job *kubeflowv1.PyTorchJob) int32 {
	jobReplicas := int32(0)
	for _, r := range job.Spec.PyTorchReplicaSpecs {
		jobReplicas += *r.Replicas
	}
	return jobReplicas
}

func genGeneralName(jobName, rtype, index string) string {
	n := jobName + "-" + rtype + "-" + index
	return strings.Replace(n, "/", "-", -1)
}

func getPortFromPyTorchJob(job *kubeflowv1.PyTorchJob, rtype commonv1.ReplicaType) (int32, error) {
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
