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

package xgboost

import (
	"fmt"
	"strconv"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

// SetPodEnv sets the pod env set for:
// - XGBoost Rabit Tracker and worker
// - LightGBM master and workers
func SetPodEnv(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	xgboostjob, ok := job.(*kubeflowv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	// Add master offset for worker pods
	if strings.EqualFold(strings.ToLower(rtype), strings.ToLower(string(kubeflowv1.XGBoostJobReplicaTypeWorker))) {
		masterSpec := xgboostjob.Spec.XGBReplicaSpecs[commonv1.ReplicaType(kubeflowv1.XGBoostJobReplicaTypeMaster)]
		masterReplicas := int(*masterSpec.Replicas)
		rank += masterReplicas
	}

	masterAddr := computeMasterAddr(xgboostjob.Name, strings.ToLower(string(kubeflowv1.XGBoostJobReplicaTypeMaster)), strconv.Itoa(0))

	masterPort, err := getPortFromXGBoostJob(xgboostjob, kubeflowv1.XGBoostJobReplicaTypeMaster)
	if err != nil {
		return err
	}

	totalReplicas := computeTotalReplicas(xgboostjob)

	var workerPort int32
	var workerAddrs []string

	if totalReplicas > 1 {
		workerPortTemp, err := getPortFromXGBoostJob(xgboostjob, kubeflowv1.XGBoostJobReplicaTypeWorker)
		if err != nil {
			return err
		}
		workerPort = workerPortTemp
		workerAddrs = make([]string, totalReplicas-1)
		for i := range workerAddrs {
			workerAddrs[i] = computeMasterAddr(xgboostjob.Name, strings.ToLower(string(kubeflowv1.XGBoostJobReplicaTypeWorker)), strconv.Itoa(i))
		}
	}

	for i := range podTemplate.Spec.Containers {
		if len(podTemplate.Spec.Containers[i].Env) == 0 {
			podTemplate.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
		}
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_PORT",
			Value: strconv.Itoa(int(masterPort)),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_ADDR",
			Value: masterAddr,
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "WORLD_SIZE",
			Value: strconv.Itoa(int(totalReplicas)),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "RANK",
			Value: strconv.Itoa(rank),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "0",
		})
		// This variables are used if it is a LightGBM job
		if totalReplicas > 1 {
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "WORKER_PORT",
				Value: strconv.Itoa(int(workerPort)),
			})
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "WORKER_ADDRS",
				Value: strings.Join(workerAddrs, ","),
			})
		}
	}

	return nil
}

func computeMasterAddr(jobName, rtype, index string) string {
	n := jobName + "-" + rtype + "-" + index
	return strings.Replace(n, "/", "-", -1)
}

// getPortFromXGBoostJob gets the port of xgboost container.
func getPortFromXGBoostJob(job *kubeflowv1.XGBoostJob, rtype commonv1.ReplicaType) (int32, error) {
	containers := job.Spec.XGBReplicaSpecs[commonv1.ReplicaType(rtype)].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == kubeflowv1.XGBoostJobDefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == kubeflowv1.XGBoostJobDefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("failed to found the port")
}

func computeTotalReplicas(obj metav1.Object) int32 {
	job := obj.(*kubeflowv1.XGBoostJob)
	jobReplicas := int32(0)

	if job.Spec.XGBReplicaSpecs == nil || len(job.Spec.XGBReplicaSpecs) == 0 {
		return jobReplicas
	}
	for _, r := range job.Spec.XGBReplicaSpecs {
		if r.Replicas == nil {
			continue
		} else {
			jobReplicas += *r.Replicas
		}
	}
	return jobReplicas
}
