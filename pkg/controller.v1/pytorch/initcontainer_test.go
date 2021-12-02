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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	pytorchv1 "github.com/kubeflow/training-operator/pkg/apis/pytorch/v1"
	"github.com/kubeflow/training-operator/pkg/config"
)

func TestInitContainer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	defer ginkgo.GinkgoRecover()

	config.Config.PyTorchInitContainerImage = config.PyTorchInitContainerImageDefault
	config.Config.PyTorchInitContainerTemplateFile = config.PyTorchInitContainerTemplateFileDefault

	testCases := []struct {
		job         *pytorchv1.PyTorchJob
		rtype       commonv1.ReplicaType
		index       string
		expected    int
		exepctedErr error
	}{
		{
			job: &pytorchv1.PyTorchJob{
				Spec: pytorchv1.PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						pytorchv1.PyTorchReplicaTypeWorker: {
							Replicas: int32Ptr(1),
						},
					},
				},
			},
			rtype:       pytorchv1.PyTorchReplicaTypeWorker,
			index:       "0",
			expected:    0,
			exepctedErr: nil,
		},
		{
			job: &pytorchv1.PyTorchJob{
				Spec: pytorchv1.PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						pytorchv1.PyTorchReplicaTypeWorker: {
							Replicas: int32Ptr(1),
						},
						pytorchv1.PyTorchReplicaTypeMaster: {
							Replicas: int32Ptr(1),
						},
					},
				},
			},
			rtype:       pytorchv1.PyTorchReplicaTypeWorker,
			index:       "0",
			expected:    1,
			exepctedErr: nil,
		},
		{
			job: &pytorchv1.PyTorchJob{
				Spec: pytorchv1.PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						pytorchv1.PyTorchReplicaTypeWorker: {
							Replicas: int32Ptr(1),
						},
						pytorchv1.PyTorchReplicaTypeMaster: {
							Replicas: int32Ptr(1),
						},
					},
				},
			},
			rtype:       pytorchv1.PyTorchReplicaTypeMaster,
			index:       "0",
			expected:    0,
			exepctedErr: nil,
		},
	}

	for _, t := range testCases {
		log := logr.Discard()
		podTemplateSpec := t.job.Spec.PyTorchReplicaSpecs[t.rtype].Template
		err := setInitContainer(t.job, &podTemplateSpec,
			strings.ToLower(string(t.rtype)), t.index, log)
		if t.exepctedErr == nil {
			gomega.Expect(err).To(gomega.BeNil())
		} else {
			gomega.Expect(err).To(gomega.Equal(t.exepctedErr))
		}
		gomega.Expect(len(podTemplateSpec.Spec.InitContainers)).To(gomega.Equal(t.expected))
	}
}
