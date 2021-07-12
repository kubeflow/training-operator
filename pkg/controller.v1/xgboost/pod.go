/*
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

package xgboost

import (
	"context"
	"fmt"
	"github.com/kubeflow/tf-operator/pkg/common/util"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	xgboostv1 "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
	corev1 "k8s.io/api/core/v1"
)

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *XGBoostJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist, client.MatchingLabels(r.GenLabels(job.GetName())))
	if err != nil {
		return nil, err
	}

	return util.ConvertPodList(podlist.Items), nil
}

// SetPodEnv sets the pod env set for:
// - XGBoost Rabit Tracker and worker
// - LightGBM master and workers
func SetPodEnv(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	xgboostjob, ok := job.(*xgboostv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	// Add master offset for worker pods
	if strings.ToLower(rtype) == strings.ToLower(string(xgboostv1.XGBoostReplicaTypeWorker)) {
		masterSpec := xgboostjob.Spec.XGBReplicaSpecs[commonv1.ReplicaType(xgboostv1.XGBoostReplicaTypeMaster)]
		masterReplicas := int(*masterSpec.Replicas)
		rank += masterReplicas
	}

	masterAddr := computeMasterAddr(xgboostjob.Name, strings.ToLower(string(xgboostv1.XGBoostReplicaTypeMaster)), strconv.Itoa(0))

	masterPort, err := GetPortFromXGBoostJob(xgboostjob, xgboostv1.XGBoostReplicaTypeMaster)
	if err != nil {
		return err
	}

	totalReplicas := computeTotalReplicas(xgboostjob)

	var workerPort int32
	var workerAddrs []string

	if totalReplicas > 1 {
		workerPortTemp, err := GetPortFromXGBoostJob(xgboostjob, xgboostv1.XGBoostReplicaTypeWorker)
		if err != nil {
			return err
		}
		workerPort = workerPortTemp
		workerAddrs = make([]string, totalReplicas-1)
		for i := range workerAddrs {
			workerAddrs[i] = computeMasterAddr(xgboostjob.Name, strings.ToLower(string(xgboostv1.XGBoostReplicaTypeWorker)), strconv.Itoa(i))
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
