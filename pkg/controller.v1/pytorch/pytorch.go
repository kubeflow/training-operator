package controllers

import (
	"fmt"
	"strconv"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	pytorchv1 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1"
	corev1 "k8s.io/api/core/v1"
)

func SetPodEnv(obj interface{}, podTemplateSpec *corev1.PodTemplateSpec, rtype, index string) error {
	pytorchjob, ok := obj.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", obj)
	}

	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	totalReplicas := getTotalReplicas(pytorchjob)

	masterPort, err := getPortFromPyTorchJob(pytorchjob, pytorchv1.PyTorchReplicaTypeMaster)
	if err != nil {
		return err
	}

	masterAddr := genGeneralName(pytorchjob.Name, strings.ToLower(string(pytorchv1.PyTorchReplicaTypeMaster)), strconv.Itoa(0))
	if rtype == string(pytorchv1.PyTorchReplicaTypeMaster) {
		if rank != 0 {
			return fmt.Errorf("invalid config: There should be only a single master with index=0")
		}
		masterAddr = "localhost"
	} else {
		rank = rank + 1
	}

	for i := range podTemplateSpec.Spec.Containers {
		if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
			podTemplateSpec.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
		}
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_PORT",
			Value: strconv.Itoa(int(masterPort)),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_ADDR",
			Value: masterAddr,
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "WORLD_SIZE",
			Value: strconv.Itoa(int(totalReplicas)),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "RANK",
			Value: strconv.Itoa(rank),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "0",
		})
	}

	return nil
}

func getTotalReplicas(job *pytorchv1.PyTorchJob) int32 {
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

func getPortFromPyTorchJob(job *pytorchv1.PyTorchJob, rtype commonv1.ReplicaType) (int32, error) {
	containers := job.Spec.PyTorchReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == pytorchv1.DefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == pytorchv1.DefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("port not found")
}
