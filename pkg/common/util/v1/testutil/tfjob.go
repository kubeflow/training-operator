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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
)

func NewTFJobWithCleanPolicy(chief, worker, ps int, policy common.CleanPodPolicy) *tfv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithCleanupJobDelay(chief, worker, ps int, ttl *int32) *tfv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.TTLSecondsAfterFinished = ttl
		policy := common.CleanPodPolicyNone
		tfJob.Spec.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.TTLSecondsAfterFinished = ttl
	policy := common.CleanPodPolicyNone
	tfJob.Spec.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithActiveDeadlineSeconds(chief, worker, ps int, ads *int64) *tfv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.ActiveDeadlineSeconds = ads
		policy := common.CleanPodPolicyAll
		tfJob.Spec.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.ActiveDeadlineSeconds = ads
	policy := common.CleanPodPolicyAll
	tfJob.Spec.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithBackoffLimit(chief, worker, ps int, backoffLimit *int32) *tfv1.TFJob {
	if chief == 1 {
		tfJob := NewTFJobWithChief(worker, ps)
		tfJob.Spec.BackoffLimit = backoffLimit
		tfJob.Spec.TFReplicaSpecs["Worker"].RestartPolicy = "OnFailure"
		policy := common.CleanPodPolicyAll
		tfJob.Spec.CleanPodPolicy = &policy
		return tfJob
	}
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.BackoffLimit = backoffLimit
	tfJob.Spec.TFReplicaSpecs["Worker"].RestartPolicy = "OnFailure"
	policy := common.CleanPodPolicyAll
	tfJob.Spec.CleanPodPolicy = &policy
	return tfJob
}

func NewTFJobWithChief(worker, ps int) *tfv1.TFJob {
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.TFReplicaSpecs[tfv1.TFReplicaTypeChief] = &common.ReplicaSpec{
		Template: NewTFReplicaSpecTemplate(),
	}
	return tfJob
}

func NewTFJobWithEvaluator(worker, ps, evaluator int) *tfv1.TFJob {
	tfJob := NewTFJob(worker, ps)
	if evaluator > 0 {
		evaluator := int32(evaluator)
		tfJob.Spec.TFReplicaSpecs[tfv1.TFReplicaTypeEval] = &common.ReplicaSpec{
			Replicas: &evaluator,
			Template: NewTFReplicaSpecTemplate(),
		}
	}
	return tfJob
}

func NewTFJob(worker, ps int) *tfv1.TFJob {
	tfJob := &tfv1.TFJob{
		TypeMeta: metav1.TypeMeta{
			Kind: tfv1.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestTFJobName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: tfv1.TFJobSpec{
			TFReplicaSpecs: make(map[tfv1.TFReplicaType]*common.ReplicaSpec),
		},
	}
	tfv1.SetObjectDefaults_TFJob(tfJob)

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &common.ReplicaSpec{
			Replicas: &worker,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[tfv1.TFReplicaTypeWorker] = workerReplicaSpec
	}

	if ps > 0 {
		ps := int32(ps)
		psReplicaSpec := &common.ReplicaSpec{
			Replicas: &ps,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[tfv1.TFReplicaTypePS] = psReplicaSpec
	}
	return tfJob
}

func NewTFJobWithNamespace(worker, ps int, ns string) *tfv1.TFJob {
	tfJob := NewTFJob(worker, ps)
	tfJob.Namespace = ns

	return tfJob
}

func NewTFJobWithEvaluatorAndNamespace(worker, ps, evaluator int, ns string) *tfv1.TFJob {
	tfJob := NewTFJobWithEvaluator(worker, ps, evaluator)
	tfJob.Namespace = ns

	return tfJob
}

func NewTFReplicaSpecTemplate() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  tfv1.DefaultContainerName,
					Image: TestImageName,
					Args:  []string{"Fake", "Fake"},
					Ports: []v1.ContainerPort{
						v1.ContainerPort{
							Name:          tfv1.DefaultPortName,
							ContainerPort: tfv1.DefaultPort,
						},
					},
				},
			},
		},
	}
}

func SetTFJobCompletionTime(tfJob *tfv1.TFJob) {
	now := metav1.Time{Time: time.Now()}
	tfJob.Status.CompletionTime = &now
}
