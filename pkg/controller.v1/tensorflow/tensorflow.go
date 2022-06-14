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
package tensorflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kubeflow/common/pkg/controller.v1/common"
	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

const (
	// EnvCustomClusterDomain is the custom defined cluster domain, such as "svc.cluster.local".
	// Ref: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#a-records
	EnvCustomClusterDomain = "CUSTOM_CLUSTER_DOMAIN"
)

// TaskSpec is the specification for a task (PS or worker) of the TFJob.
type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// ClusterSpec represents a cluster TensorFlow specification.
// https://www.tensorflow.org/deploy/distributed#create_a_tftrainclusterspec_to_describe_the_cluster
// It is a map from job names to network addresses.
type ClusterSpec map[string][]string

// TFConfig is a struct representing the distributed TensorFlow config.
// This struct is turned into an environment variable TF_CONFIG
// which is used by TensorFlow processes to configure themselves.
// https://www.tensorflow.org/api_docs/python/tf/estimator/RunConfig#methods
// https://cloud.google.com/ml-engine/docs/tensorflow/distributed-training-details
type TFConfig struct {
	// Cluster represents a TensorFlow ClusterSpec.
	// See: https://www.tensorflow.org/api_docs/python/tf/train/ClusterSpec
	Cluster ClusterSpec `json:"cluster"`
	Task    TaskSpec    `json:"task"`
	// Environment is used by tensorflow.contrib.learn.python.learn in versions <= 1.3
	// TODO(jlewi): I don't think it is used in versions TF >- 1.4. So we can eventually get rid of it.
	Environment string `json:"environment"`
}

// SparseClusterSpec enables a server to be configured without needing to know
// the identity of (for example) all other worker tasks.
// https://www.tensorflow.org/api_docs/python/tf/train/ClusterSpec
type SparseClusterSpec struct {
	Worker map[int32]string `json:"worker"`
	PS     []string         `json:"ps"`
}

type SparseTFConfig struct {
	Cluster SparseClusterSpec `json:"sparseCluster"`
	Task    TaskSpec          `json:"task"`
}

func convertClusterSpecToSparseClusterSpec(clusterSpec ClusterSpec, rtype string, index int32) SparseClusterSpec {
	sparseClusterSpec := SparseClusterSpec{Worker: map[int32]string{}, PS: []string{}}
	if rtype == strings.ToLower(string(kubeflowv1.TFJobReplicaTypePS)) {
		sparseClusterSpec.PS = append(sparseClusterSpec.PS, clusterSpec[rtype][index])
	} else if rtype == strings.ToLower(string(kubeflowv1.TFJobReplicaTypeWorker)) {
		sparseClusterSpec.PS = clusterSpec[strings.ToLower(string(kubeflowv1.TFJobReplicaTypePS))]
		sparseClusterSpec.Worker[index] = clusterSpec[rtype][index]
	}
	return sparseClusterSpec
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
func genTFConfigJSONStr(tfjob *kubeflowv1.TFJob, rtype, index string) (string, error) {
	// Configure the TFCONFIG environment variable.
	i, err := strconv.ParseInt(index, 0, 32)
	if err != nil {
		return "", err
	}

	cluster, err := genClusterSpec(tfjob)
	if err != nil {
		return "", err
	}

	var tfConfigJSONByteSlice []byte
	if tfjob.Spec.EnableDynamicWorker {
		sparseCluster := convertClusterSpecToSparseClusterSpec(cluster, strings.ToLower(rtype), int32(i))
		sparseTFConfig := SparseTFConfig{
			Cluster: sparseCluster,
			Task: TaskSpec{
				Type:  strings.ToLower(rtype),
				Index: int(i),
			},
		}
		tfConfigJSONByteSlice, err = json.Marshal(sparseTFConfig)
	} else {
		tfConfig := TFConfig{
			Cluster: cluster,
			Task: TaskSpec{
				Type:  strings.ToLower(rtype),
				Index: int(i),
			},
			// We need to set environment to cloud  otherwise it will default to local which isn't what we want.
			// Environment is used by tensorflow.contrib.learn.python.learn in versions <= 1.3
			// TODO(jlewi): I don't think it is used in versions TF >- 1.4. So we can eventually get rid of it.
			Environment: "cloud",
		}
		tfConfigJSONByteSlice, err = json.Marshal(tfConfig)
	}
	if err != nil {
		return "", err
	}

	return string(tfConfigJSONByteSlice), nil
}

// genClusterSpec will generate ClusterSpec.
func genClusterSpec(tfjob *kubeflowv1.TFJob) (ClusterSpec, error) {
	clusterSpec := make(ClusterSpec)

	for rtype, spec := range tfjob.Spec.TFReplicaSpecs {
		rt := strings.ToLower(string(rtype))
		replicaNames := make([]string, 0, *spec.Replicas)

		port, err := GetPortFromTFJob(tfjob, rtype)
		if err != nil {
			return nil, err
		}
		for i := int32(0); i < *spec.Replicas; i++ {
			// As described here: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#a-records.
			// Headless service assigned a DNS A record for a name of the form "my-svc.my-namespace.svc.cluster.local".
			// And the last part "svc.cluster.local" is called cluster domain
			// which maybe different between kubernetes clusters.
			hostName := common.GenGeneralName(tfjob.Name, rt, fmt.Sprintf("%d", i))
			svcName := hostName + "." + tfjob.Namespace + "." + "svc"
			clusterDomain := os.Getenv(EnvCustomClusterDomain)
			if len(clusterDomain) > 0 {
				svcName += "." + clusterDomain
			}

			endpoint := fmt.Sprintf("%s:%d", svcName, port)
			replicaNames = append(replicaNames, endpoint)
		}

		clusterSpec[rt] = replicaNames
	}

	return clusterSpec, nil
}
