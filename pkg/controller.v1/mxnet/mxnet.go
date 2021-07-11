package mxnet

import (
	"fmt"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	mxnetv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	"strconv"
	"strings"
)

const (
	// Label is used as tunerServerKey, it's designed for tvm auto-tuning.
	mxJobTunerServerKey = "tuner-server-key"
)

// MXConfig is a struct representing the distributed Mxnet config.
// This struct is turned into an environment variable MX_CONFIG
// which is used by Mxnet processes to configure themselves.
type MXConfig struct {
	// Cluster represents a Mxnet ClusterSpec.
	Cluster ClusterSpec `json:"cluster"`
	// Labels include all label of task.
	Labels LabelsSpec `json:"labels"`
	// Task include information of current node.
	Task TaskSpec `json:"task"`
}

// ClusterSpec represents a cluster Mxnet specification.
type ClusterSpec map[string][]UrlPort

type UrlPort struct {
	Url  string `json:"url"`
	Port int    `json:"port"`
}

// LabelsSpec represents a label specification.
type LabelsSpec map[string]string

// TaskSpec is the specification for a task (server or worker ...) of the MXJob.
type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

func genMXConfig(mxjob *mxnetv1.MXJob, rtype, index string) (MXConfig, error) {
	// Configure the MXCONFIG environment variable.
	i, err := strconv.ParseInt(index, 0, 32)
	if err != nil {
		return MXConfig{}, err
	}

	cluster, err := genClusterSpec(mxjob)
	if err != nil {
		return MXConfig{}, err
	}

	labels, err := genLabelsSpec(mxjob)
	if err != nil {
		return MXConfig{}, err
	}

	mxConfig := MXConfig{
		Cluster: cluster,
		Labels:  labels,
		Task: TaskSpec{
			Type:  rtype,
			Index: int(i),
		},
	}

	return mxConfig, nil
}

// genClusterSpec will generate ClusterSpec.
func genClusterSpec(mxjob *mxnetv1.MXJob) (ClusterSpec, error) {
	clusterSpec := make(ClusterSpec)

	for rtype, spec := range mxjob.Spec.MXReplicaSpecs {
		rt := strings.ToLower(string(rtype))
		replicaNames := make([]UrlPort, 0, *spec.Replicas)

		port, err := GetPortFromMXJob(mxjob, rtype)
		if err != nil {
			return nil, err
		}
		for i := int32(0); i < *spec.Replicas; i++ {
			host := UrlPort{
				Url:  common.GenGeneralName(mxjob.Name, rt, fmt.Sprintf("%d", i)),
				Port: int(port),
			}
			replicaNames = append(replicaNames, host)
		}

		clusterSpec[rt] = replicaNames
	}

	return clusterSpec, nil
}

// genLabelsSpec will generate LabelsSpec.
func genLabelsSpec(mxjob *mxnetv1.MXJob) (LabelsSpec, error) {
	labelsSpec := make(LabelsSpec)

	for rtype, spec := range mxjob.Spec.MXReplicaSpecs {
		rt := strings.ToLower(string(rtype))

		labelsSpec[rt] = spec.Template.Annotations[mxJobTunerServerKey]
	}

	return labelsSpec, nil
}
