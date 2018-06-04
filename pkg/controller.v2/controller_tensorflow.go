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

// Package controller provides a Kubernetes controller for a TFJob resource.
package controller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

// TFConfig is a struct representing the distributed TensorFlow config.
// This struct is turned into an environment variable TF_CONFIG
// which is used by TensorFlow processes to configure themselves.
// https://cloud.google.com/ml-engine/docs/trainer-considerations#use_tf_config
type TFConfig struct {
	// Cluster represents a TensorFlow ClusterSpec.
	// See: https://www.tensorflow.org/api_docs/python/tf/train/ClusterSpec
	Cluster ClusterSpec `json:"cluster"`
	Task    TaskSpec    `json:"task"`
}

// ClusterSpec represents a cluster TensorFlow specification.
// https://www.tensorflow.org/deploy/distributed#create_a_tftrainclusterspec_to_describe_the_cluster
// It is a map from job names to network addresses.
type ClusterSpec map[string][]string

// TaskSpec is the specification for a task (PS or worker) of the TFJob.
type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// genTFConfig will generate the environment variable TF_CONFIG
// {
//     "cluster": {
//         "ps": ["ps1:2222", "ps2:2222"],
//         "worker": ["worker1:2222", "worker2:2222", "worker3:2222"]
//     },
//     "task": {
//         "type": "ps",
//         "index": 1
//         },
//     }
// }
func genTFConfigJSONStr(tfjob *tfv1alpha2.TFJob, rtype, index string) string {
	// Configure the TFCONFIG environment variable.
	i, _ := strconv.ParseInt(index, 0, 32)

	tfConfig := TFConfig{
		Cluster: genClusterSpec(tfjob),
		Task: TaskSpec{
			Type:  rtype,
			Index: int(i),
		},
	}

	tfConfigJSONStr, err := json.Marshal(tfConfig)
	if err != nil {
		log.Errorf("TFJob: %v serializing tfConfig return error: %v", tfjob.Name, err)
		return ""
	}

	return string(tfConfigJSONStr)
}

// genClusterSpec will generate ClusterSpec.
func genClusterSpec(tfjob *tfv1alpha2.TFJob) ClusterSpec {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfjob, err))
		return nil
	}

	clusterSpec := make(ClusterSpec)

	for rtype, spec := range tfjob.Spec.TFReplicaSpecs {
		rt := strings.ToLower(string(rtype))
		replicaNames := make([]string, 0, *spec.Replicas)

		for i := int32(0); i < *spec.Replicas; i++ {
			host := genDNSRecord(tfjobKey, rt, fmt.Sprintf("%d", i), tfjob.ObjectMeta.Namespace) + ":" + defaultPortStr
			replicaNames = append(replicaNames, host)
		}

		clusterSpec[rt] = replicaNames
	}

	return clusterSpec
}
