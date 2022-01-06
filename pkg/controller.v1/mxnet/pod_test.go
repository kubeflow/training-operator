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

package mxnet

import (
	"strings"
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mxv1 "github.com/kubeflow/training-operator/pkg/apis/mxnet/v1"
	"github.com/kubeflow/training-operator/pkg/common/util/v1/testutil"
	corev1 "k8s.io/api/core/v1"
)

func setRestartPolicy(podTemplateSpec *corev1.PodTemplateSpec, spec *commonv1.ReplicaSpec) {
	if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicy(spec.RestartPolicy)
	}
}

func TestRestartPolicy(t *testing.T) {
	type tc struct {
		mxJob                 *mxv1.MXJob
		expectedRestartPolicy corev1.RestartPolicy
		expectedType          commonv1.ReplicaType
	}
	testCase := []tc{
		func() tc {
			mxJob := testutil.NewMXJob(1, 0)
			specRestartPolicy := commonv1.RestartPolicyExitCode
			mxJob.Spec.MXReplicaSpecs[mxv1.MXReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				mxJob:                 mxJob,
				expectedRestartPolicy: corev1.RestartPolicyNever,
				expectedType:          mxv1.MXReplicaTypeWorker,
			}
		}(),
		func() tc {
			mxJob := testutil.NewMXJob(1, 0)
			specRestartPolicy := commonv1.RestartPolicyNever
			mxJob.Spec.MXReplicaSpecs[mxv1.MXReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				mxJob:                 mxJob,
				expectedRestartPolicy: corev1.RestartPolicyNever,
				expectedType:          mxv1.MXReplicaTypeWorker,
			}
		}(),
		func() tc {
			mxJob := testutil.NewMXJob(1, 0)
			specRestartPolicy := commonv1.RestartPolicyAlways
			mxJob.Spec.MXReplicaSpecs[mxv1.MXReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				mxJob:                 mxJob,
				expectedRestartPolicy: corev1.RestartPolicyAlways,
				expectedType:          mxv1.MXReplicaTypeWorker,
			}
		}(),
		func() tc {
			mxJob := testutil.NewMXJob(1, 0)
			specRestartPolicy := commonv1.RestartPolicyOnFailure
			mxJob.Spec.MXReplicaSpecs[mxv1.MXReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				mxJob:                 mxJob,
				expectedRestartPolicy: corev1.RestartPolicyOnFailure,
				expectedType:          mxv1.MXReplicaTypeWorker,
			}
		}(),
	}
	for _, c := range testCase {
		spec := c.mxJob.Spec.MXReplicaSpecs[c.expectedType]
		podTemplate := spec.Template
		setRestartPolicy(&podTemplate, spec)
		if podTemplate.Spec.RestartPolicy != c.expectedRestartPolicy {
			t.Errorf("Expected %s, got %s", c.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
		}
	}
}

func TestBytPSEnv(t *testing.T) {
	type tc struct {
		container   *corev1.Container
		rtype       commonv1.ReplicaType
		index       string
		expectedVal string
	}
	testCase := []tc{
		{
			container:   &corev1.Container{},
			rtype:       mxv1.MXReplicaTypeWorker,
			index:       "0",
			expectedVal: "0",
		},
		{
			container:   &corev1.Container{},
			rtype:       mxv1.MXReplicaTypeWorker,
			index:       "1",
			expectedVal: "1",
		},
		{
			container:   &corev1.Container{},
			rtype:       mxv1.MXReplicaTypeScheduler,
			index:       "0",
			expectedVal: "",
		},
		{
			container:   &corev1.Container{},
			rtype:       mxv1.MXReplicaTypeServer,
			index:       "0",
			expectedVal: "",
		},
		{
			container:   &corev1.Container{},
			rtype:       mxv1.MXReplicaTypeTunerTracker,
			index:       "0",
			expectedVal: "",
		},
		{
			container:   &corev1.Container{},
			rtype:       mxv1.MXReplicaTypeTunerServer,
			index:       "0",
			expectedVal: "",
		},
		{
			container:   &corev1.Container{},
			rtype:       mxv1.MXReplicaTypeTuner,
			index:       "0",
			expectedVal: "",
		},
	}

	for _, c := range testCase {
		addBytePSEnv(c.container, strings.ToLower(string(c.rtype)), c.index)
		var val string
		for _, env := range c.container.Env {
			if env.Name == "DMLC_WORKER_ID" {
				val = env.Value
				break
			}
		}

		if val != c.expectedVal {
			t.Errorf("Expected %s, got %s", c.expectedVal, val)
		}
	}
}
