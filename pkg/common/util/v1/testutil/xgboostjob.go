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
// limitations under the License.

package testutil

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xgboostjobv1 "github.com/kubeflow/training-operator/pkg/apis/xgboost/v1"
)

func NewXGBoostJobWithMaster(worker int) *xgboostjobv1.XGBoostJob {
	job := NewXGoostJob(worker)
	master := int32(1)
	masterReplicaSpec := &commonv1.ReplicaSpec{
		Replicas: &master,
		Template: NewXGBoostReplicaSpecTemplate(),
	}
	job.Spec.XGBReplicaSpecs[commonv1.ReplicaType(xgboostjobv1.XGBoostReplicaTypeMaster)] = masterReplicaSpec
	return job
}

func NewXGoostJob(worker int) *xgboostjobv1.XGBoostJob {

	job := &xgboostjobv1.XGBoostJob{
		TypeMeta: metav1.TypeMeta{
			Kind: xgboostjobv1.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-xgboostjob",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: xgboostjobv1.XGBoostJobSpec{
			XGBReplicaSpecs: make(map[commonv1.ReplicaType]*commonv1.ReplicaSpec),
		},
	}

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &commonv1.ReplicaSpec{
			Replicas: &worker,
			Template: NewXGBoostReplicaSpecTemplate(),
		}
		job.Spec.XGBReplicaSpecs[commonv1.ReplicaType(xgboostjobv1.XGBoostReplicaTypeWorker)] = workerReplicaSpec
	}

	return job
}

func NewXGBoostReplicaSpecTemplate() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  xgboostjobv1.DefaultContainerName,
					Image: "test-image-for-kubeflow-xgboost-operator:latest",
					Args:  []string{"Fake", "Fake"},
					Ports: []corev1.ContainerPort{
						corev1.ContainerPort{
							Name:          xgboostjobv1.DefaultPortName,
							ContainerPort: xgboostjobv1.DefaultPort,
						},
					},
				},
			},
		},
	}
}
