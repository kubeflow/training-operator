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

package v1alpha1

import (
	"github.com/golang/protobuf/proto"
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_TFJob initializes any uninitialized values to default values
func SetDefaults_TFJob(obj *TFJob) {
	c := &obj.Spec

	if c.TFImage == "" {
		c.TFImage = DefaultTFImage
	}

	// Check that each replica has a TensorFlow container.
	for _, r := range c.ReplicaSpecs {

		if r.TFPort == nil {
			r.TFPort = proto.Int32(TFPort)
		}

		if string(r.TFReplicaType) == "" {
			r.TFReplicaType = MASTER
		}

		if r.Replicas == nil {
			r.Replicas = proto.Int32(Replicas)
		}
	}
	if c.TerminationPolicy == nil {
		c.TerminationPolicy = &TerminationPolicySpec{
			Chief: &ChiefSpec{
				ReplicaName:  "MASTER",
				ReplicaIndex: 0,
			},
		}
	}

}
