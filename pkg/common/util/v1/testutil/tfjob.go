// Copyright 2018 The Kubeflow Authors
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
// limitations under the License.

package testutil

import (
	"time"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func NewTFJobWithCleanPolicy(chief, worker, ps int, policy commonv1.CleanPodPolicy) *kubeflowv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithCleanupJobDelay(chief, worker, ps int, ttl *int32) *kubeflowv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.RunPolicy.TTLSecondsAfterFinished = ttl
		policy := commonv1.CleanPodPolicyNone
		tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.RunPolicy.TTLSecondsAfterFinished = ttl
	policy := commonv1.CleanPodPolicyNone
	tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithActiveDeadlineSeconds(chief, worker, ps int, ads *int64) *kubeflowv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.RunPolicy.ActiveDeadlineSeconds = ads
		policy := commonv1.CleanPodPolicyAll
		tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.RunPolicy.ActiveDeadlineSeconds = ads
	policy := commonv1.CleanPodPolicyAll
	tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithBackoffLimit(chief, worker, ps int, backoffLimit *int32) *kubeflowv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.RunPolicy.BackoffLimit = backoffLimit
		tfJob.Spec.TFReplicaSpecs["Worker"].RestartPolicy = "OnFailure"
		policy := commonv1.CleanPodPolicyAll
		tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.RunPolicy.BackoffLimit = backoffLimit
	tfJob.Spec.TFReplicaSpecs["Worker"].RestartPolicy = "OnFailure"
	policy := commonv1.CleanPodPolicyAll
	tfJob.Spec.RunPolicy.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithChief(worker, ps int) *kubeflowv1.TFJob {
	tfJob := NewTFJob(worker, ps)
	chief := int32(1)
	tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeChief] = &commonv1.ReplicaSpec{
		Replicas: &chief,
		Template: NewTFReplicaSpecTemplate(),
	}
	return tfJob
}

func NewTFJobWithEvaluator(worker, ps, evaluator int) *kubeflowv1.TFJob {
	tfJob := NewTFJob(worker, ps)
	if evaluator > 0 {
		evaluator := int32(evaluator)
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeEval] = &commonv1.ReplicaSpec{
			Replicas: &evaluator,
			Template: NewTFReplicaSpecTemplate(),
		}
	}
	return tfJob
}

func NewTFJobWithSuccessPolicy(worker, ps int, successPolicy kubeflowv1.SuccessPolicy) *kubeflowv1.TFJob {
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.SuccessPolicy = &successPolicy
	return tfJob
}

func NewTFJob(worker, ps int) *kubeflowv1.TFJob {
	tfJob := &kubeflowv1.TFJob{
		TypeMeta: metav1.TypeMeta{
			Kind: TFJobKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestTFJobName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: kubeflowv1.TFJobSpec{
			TFReplicaSpecs: make(map[commonv1.ReplicaType]*commonv1.ReplicaSpec),
		},
	}
	kubeflowv1.SetObjectDefaults_TFJob(tfJob)

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &worker,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker] = workerReplicaSpec
	}

	if ps > 0 {
		ps := int32(ps)
		psReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &ps,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypePS] = psReplicaSpec
	}
	return tfJob
}

func NewTFJobV2(worker, ps, master, chief, evaluator int) *kubeflowv1.TFJob {
	tfJob := &kubeflowv1.TFJob{
		TypeMeta: metav1.TypeMeta{
			Kind: TFJobKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestTFJobName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: kubeflowv1.TFJobSpec{
			TFReplicaSpecs: make(map[commonv1.ReplicaType]*commonv1.ReplicaSpec),
		},
	}
	kubeflowv1.SetObjectDefaults_TFJob(tfJob)

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &worker,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker] = workerReplicaSpec
	}

	if ps > 0 {
		ps := int32(ps)
		psReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &ps,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypePS] = psReplicaSpec
	}

	if master > 0 {
		master := int32(master)
		masterReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &master,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeMaster] = masterReplicaSpec
	}

	if chief > 0 {
		chief := int32(chief)
		chiefReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &chief,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeChief] = chiefReplicaSpec
	}

	if evaluator > 0 {
		evaluator := int32(evaluator)
		evaluatorReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &evaluator,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeChief] = evaluatorReplicaSpec
	}
	return tfJob
}

func NewTFJobWithNamespace(worker, ps int, ns string) *kubeflowv1.TFJob {
	tfJob := NewTFJob(worker, ps)
	tfJob.Namespace = ns

	return tfJob
}

func NewTFJobWithEvaluatorAndNamespace(worker, ps, evaluator int, ns string) *kubeflowv1.TFJob {
	tfJob := NewTFJobWithEvaluator(worker, ps, evaluator)
	tfJob.Namespace = ns

	return tfJob
}

func NewTFReplicaSpecTemplate() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  kubeflowv1.TFJobDefaultContainerName,
					Image: TestImageName,
					Args:  []string{"Fake", "Fake"},
					Ports: []v1.ContainerPort{
						v1.ContainerPort{
							Name:          kubeflowv1.TFJobDefaultPortName,
							ContainerPort: kubeflowv1.TFJobDefaultPort,
						},
					},
				},
			},
		},
	}
}

func SetTFJobCompletionTime(tfJob *kubeflowv1.TFJob) {
	now := metav1.Time{Time: time.Now()}
	tfJob.Status.CompletionTime = &now
}
