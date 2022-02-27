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
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestElasticGenerate(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	defer ginkgo.GinkgoRecover()

	backendC10D := kubeflowv1.BackendC10D

	tests := []struct {
		name        string
		job         *kubeflowv1.PyTorchJob
		expectedErr error
		expected    []corev1.EnvVar
	}{
		{
			name: "Without ElasticPolicy",
			job: &kubeflowv1.PyTorchJob{
				Spec: kubeflowv1.PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.PyTorchJobReplicaTypeWorker: {
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			expectedErr: nil,
			expected:    nil,
		},
		{
			name: "With ElasticPolicy",
			job: &kubeflowv1.PyTorchJob{
				Spec: kubeflowv1.PyTorchJobSpec{
					ElasticPolicy: &kubeflowv1.ElasticPolicy{
						MinReplicas: pointer.Int32(1),
						MaxReplicas: pointer.Int32(3),
						RDZVBackend: &backendC10D,
						RDZVPort:    pointer.Int32(1234),
						RDZVHost:    pointer.String("localhost"),
						RDZVID:      pointer.String("rdzv-id"),
						RDZVConf: []kubeflowv1.RDZVConf{
							{
								Key:   "rdzv-conf-name",
								Value: "rdzv-conf-value",
							},
							{
								Key:   "rdzv-conf-name-1",
								Value: "rdzv-conf-value-1",
							},
						},
						NProcPerNode: pointer.Int32(1),
						MaxRestarts:  pointer.Int32(3),
					},
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.PyTorchJobReplicaTypeWorker: {
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			expectedErr: nil,
			expected: []corev1.EnvVar{
				{
					Name:  EnvMaxRestarts,
					Value: "3",
				},
				{
					Name:  EnvNProcPerNode,
					Value: "1",
				},
				{
					Name:  EnvRDZVBackend,
					Value: "c10d",
				},
				{
					Name:  EnvRDZVEndpoint,
					Value: "localhost:1234",
				},
				{
					Name:  EnvRDZVID,
					Value: "rdzv-id",
				},
				{
					Name:  EnvRDZVConf,
					Value: "rdzv-conf-name=rdzv-conf-value,rdzv-conf-name-1=rdzv-conf-value-1",
				},
				{
					Name:  EnvNNodes,
					Value: "1:3",
				},
			},
		},
	}

	for _, test := range tests {
		actual, err := GetElasticEnvVarGenerator().Generate(test.job)
		if test.expectedErr == nil {
			gomega.Expect(err).To(gomega.BeNil())
		} else {
			gomega.Expect(err).To(gomega.Equal(test.expectedErr))
		}
		if test.expected == nil {
			gomega.Expect(actual).To(gomega.BeNil())
		} else {
			gomega.Expect(actual).To(gomega.ConsistOf(test.expected))
		}
	}
}
