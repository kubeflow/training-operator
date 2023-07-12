// Copyright 2022 The Kubeflow Authors
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

package paddle

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

const (
	EnvMasterEndpoint = "PADDLE_MASTER"
	EnvNumNodes       = "PADDLE_NNODES"
	EnvJobID          = "PADDLE_JOB_ID"
	EnvServerNum      = "PADDLE_SERVER_NUM"
	EnvTrainerNum     = "PADDLE_TRAINER_NUM"
)

// EnvVarGenerator is the environment variable generator interface.
type EnvVarGenerator interface {
	Generate(job *kubeflowv1.PaddleJob) ([]corev1.EnvVar, error)
}

func setPodEnv(obj interface{}, podTemplateSpec *corev1.PodTemplateSpec, rtype, index string) error {
	paddlejob, ok := obj.(*kubeflowv1.PaddleJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PaddleJob", obj)
	}

	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	totalReplicas := getTotalReplicas(paddlejob)

	for i := range podTemplateSpec.Spec.Containers {
		// Initialize the environment variables.
		if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
			podTemplateSpec.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
		}
		// Set PYTHONUNBUFFERED to true, to disable output buffering.
		// Ref https://stackoverflow.com/questions/59812009/what-is-the-use-of-pythonunbuffered-in-docker-file.
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "1",
		})

		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  EnvJobID,
			Value: paddlejob.Name,
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  EnvNumNodes,
			Value: strconv.Itoa(int(totalReplicas)),
		})

		// If the master is null, run in Collective mode
		if paddlejob.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeMaster] == nil {

			// We pick the worker 0 as the rendezvous endpoint
			masterAddr := replicaName(paddlejob.Name, kubeflowv1.PaddleJobReplicaTypeWorker, 0)
			masterPort := getPortFromPaddleJob(paddlejob, kubeflowv1.PaddleJobReplicaTypeWorker)
			if rank == 0 {
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name: "POD_IP_DUMMY",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "status.podIP",
						},
					},
				})
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  EnvMasterEndpoint,
					Value: fmt.Sprintf("$(POD_IP_DUMMY):%d", masterPort),
				})
			} else {
				// NOTE(kuizhiqing): no need to ensure master ready by initcontainer or alternative methods, paddle launch will handle it.
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  EnvMasterEndpoint,
					Value: fmt.Sprintf("%s:%d", masterAddr, masterPort),
				})
			}

			// Otherwise, run in PS mode
		} else {

			// We pick the master 0 as the rendezvous endpoint
			masterAddr := replicaName(paddlejob.Name, kubeflowv1.PaddleJobReplicaTypeMaster, 0)
			masterPort := getPortFromPaddleJob(paddlejob, kubeflowv1.PaddleJobReplicaTypeMaster)
			if rank == 0 && rtype == strings.ToLower(string(kubeflowv1.PaddleJobReplicaTypeMaster)) {
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name: "POD_IP_DUMMY",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "status.podIP",
						},
					},
				})
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  EnvMasterEndpoint,
					Value: fmt.Sprintf("$(POD_IP_DUMMY):%d", masterPort),
				})
			} else {
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  EnvMasterEndpoint,
					Value: fmt.Sprintf("%s:%d", masterAddr, masterPort),
				})
			}

			// Each pod will have only one server or trainer.
			if rtype == strings.ToLower(string(kubeflowv1.PaddleJobReplicaTypeMaster)) {
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  EnvServerNum,
					Value: "1",
				})
			} else {
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  EnvTrainerNum,
					Value: "1",
				})
			}

		}
	}

	return nil
}

func getTotalReplicas(job *kubeflowv1.PaddleJob) int32 {
	jobReplicas := int32(0)
	for _, r := range job.Spec.PaddleReplicaSpecs {
		jobReplicas += *r.Replicas
	}
	return jobReplicas
}

func replicaName(jobName string, rtype kubeflowv1.ReplicaType, index int) string {
	n := jobName + "-" + strings.ToLower(string(rtype)) + "-" + strconv.Itoa(index)
	return strings.Replace(n, "/", "-", -1)
}

func getPortFromPaddleJob(job *kubeflowv1.PaddleJob, rtype kubeflowv1.ReplicaType) int32 {
	containers := job.Spec.PaddleReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == kubeflowv1.PaddleJobDefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == kubeflowv1.PaddleJobDefaultPortName {
					return port.ContainerPort
				}
			}
		}
	}
	return kubeflowv1.PaddleJobDefaultPort
}
