// Copyright 2024 The Kubeflow Authors
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

package jax

import (
	"errors"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

var (
	errorDefaulContainerPortNotExposed = errors.New("default container port is not exposed")
	errorFailedToRecognizeRank         = errors.New("failed to recognize the JAXJob Rank")
)

type EnvVarGenerator interface {
	Generate(job *kubeflowv1.JAXJob) ([]corev1.EnvVar, error)
}

func setPodEnv(jaxjob *kubeflowv1.JAXJob, podTemplateSpec *corev1.PodTemplateSpec, rtype, index string) error {

	coordinatorAddr := replicaName(jaxjob.Name, kubeflowv1.JAXJobReplicaTypeWorker, 0)

	coordinatorPort, err := getPortFromJAXJob(jaxjob, kubeflowv1.JAXJobReplicaTypeWorker)
	if err != nil {
		return err
	}

	totalReplicas := getTotalReplicas(jaxjob)

	for i := range podTemplateSpec.Spec.Containers {

		rank, err := strconv.Atoi(index)
		if err != nil {
			return errorFailedToRecognizeRank
		}
		// Set PYTHONUNBUFFERED to true, to disable output buffering.
		// Ref https://stackoverflow.com/questions/59812009/what-is-the-use-of-pythonunbuffered-in-docker-file.
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "1",
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "COORDINATOR_PORT",
			Value: strconv.Itoa(int(coordinatorPort)),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "COORDINATOR_ADDRESS",
			Value: coordinatorAddr,
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "NUM_PROCESSES",
			Value: strconv.Itoa(int(totalReplicas)),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PROCESS_ID",
			Value: strconv.Itoa(rank),
		})
	}

	return nil
}

func getTotalReplicas(job *kubeflowv1.JAXJob) int {
	jobReplicas := 0
	for _, r := range job.Spec.JAXReplicaSpecs {
		jobReplicas += int(ptr.Deref[int32](r.Replicas, 0))
	}
	return jobReplicas
}

func replicaName(jobName string, rtype kubeflowv1.ReplicaType, index int) string {
	n := jobName + "-" + strings.ToLower(string(rtype)) + "-" + strconv.Itoa(index)
	return strings.Replace(n, "/", "-", -1)
}

func getPortFromJAXJob(job *kubeflowv1.JAXJob, rtype kubeflowv1.ReplicaType) (int32, error) {
	containers := job.Spec.JAXReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == kubeflowv1.JAXJobDefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == kubeflowv1.JAXJobDefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, errorDefaulContainerPortNotExposed
}
