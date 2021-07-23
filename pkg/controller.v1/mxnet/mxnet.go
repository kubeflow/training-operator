package mxnet

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	mxnetv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// Label is used as tunerServerKey, it's designed for tvm auto-tuning.
	mxJobTunerServerKey = "tuner-server-key"
	// mxConfig is the environment variable name of MXNet cluster spec.
	mxConfig = "MX_CONFIG"
)

var (
	errPortNotFound = fmt.Errorf("failed to found the port")
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

func SetPodEnv(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	mxJob, ok := job.(*mxnetv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	// Generate MX_CONFIG JSON.
	mxConfigData, err := genMXConfig(mxJob, rtype, index)
	if err != nil {
		return err
	}

	// Generate MX_CONFIG JSON Str.
	mxConfigJson, err := json.Marshal(mxConfigData)
	if err != nil {
		return err
	}

	// Add MX_CONFIG environment variable.
	for i := range podTemplate.Spec.Containers {

		c := &podTemplate.Spec.Containers[i]

		// Set environment variable MX_CONFIG
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  mxConfig,
			Value: string(mxConfigJson),
		})

		// Set Mxnet Distributed Training environment variable
		// We get these envs from MX_COFING to make them stay identical
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_PS_ROOT_PORT",
			Value: strconv.Itoa(getConfigAddr(&mxConfigData, mxnetv1.MXReplicaTypeScheduler, 0).Port),
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_PS_ROOT_URI",
			Value: getConfigAddr(&mxConfigData, mxnetv1.MXReplicaTypeScheduler, 0).Url,
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_NUM_SERVER",
			Value: strconv.Itoa(getConfigReplica(&mxConfigData, mxnetv1.MXReplicaTypeServer)),
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_NUM_WORKER",
			Value: strconv.Itoa(getConfigReplica(&mxConfigData, mxnetv1.MXReplicaTypeWorker)),
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_ROLE",
			Value: mxConfigData.Task.Type,
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_USE_KUBERNETES",
			Value: strconv.Itoa(1),
		})

		// BytePS needs env DMLC_WORKER_ID for each worker
		addBytePSEnv(c, rtype, index)
	}
	return nil
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

		port, err := getPortFromMXJob(mxjob, rtype)
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

func getConfigAddr(mxConfigData *MXConfig, rtype commonv1.ReplicaType, index int) UrlPort {
	rt := strings.ToLower(string(rtype))
	var url_port UrlPort
	if len(mxConfigData.Cluster[rt]) <= index {
		// index out of range, maybe this url doen't exist
		url_port = UrlPort{
			Url:  "",
			Port: 0,
		}
	} else {
		url_port = mxConfigData.Cluster[rt][index]
	}
	return url_port
}

func getConfigReplica(mxConfigData *MXConfig, rtype commonv1.ReplicaType) int {
	rt := strings.ToLower(string(rtype))
	return len(mxConfigData.Cluster[rt])
}

// getPortFromMXJob gets the port of mxnet container.
func getPortFromMXJob(mxJob *mxnetv1.MXJob, rtype commonv1.ReplicaType) (int32, error) {
	containers := mxJob.Spec.MXReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == mxnetv1.DefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == mxnetv1.DefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, errPortNotFound
}

func addBytePSEnv(c *corev1.Container, rtype, index string) {
	if rtype == strings.ToLower(string(mxnetv1.MXReplicaTypeWorker)) {
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_WORKER_ID",
			Value: index,
		})
	}
}

func setRestartPolicy(podTemplateSpec *corev1.PodTemplateSpec, spec *commonv1.ReplicaSpec) {
	if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicy(spec.RestartPolicy)
	}
}

func ContainSchedulerSpec(mxJob *mxnetv1.MXJob) bool {
	if _, ok := mxJob.Spec.MXReplicaSpecs[mxnetv1.MXReplicaTypeScheduler]; ok {
		return true
	}
	return false
}
