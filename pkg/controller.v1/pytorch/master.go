package pytorch

import (
	"strconv"
	"strings"
	"sync"

	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/training/v1"

	corev1 "k8s.io/api/core/v1"
)

var (
	masterGenerator EnvVarGenerator
	onceMaster      sync.Once
	EnvMasterPort   = "MASTER_PORT"
	EnvMasterAddr   = "MASTER_ADDR"
)

// MasterEnvVarGenerator is the environment variable generator for Master related arguments.
type MasterEnvVarGenerator struct {
}

func GetMasterEnvVarGenerator() EnvVarGenerator {
	onceMaster.Do(func() {
		masterGenerator = &MasterEnvVarGenerator{}
	})
	return masterGenerator
}

func (e MasterEnvVarGenerator) Generate(
	job *trainingv1.PyTorchJob) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}
	if job.Spec.PyTorchReplicaSpecs[trainingv1.PyTorchReplicaTypeMaster] != nil {
		masterPort, err := getPortFromPyTorchJob(job, trainingv1.PyTorchReplicaTypeMaster)
		if err != nil {
			return nil, err
		}

		masterAddr := genGeneralName(job.Name, strings.ToLower(string(trainingv1.PyTorchReplicaTypeMaster)), strconv.Itoa(0))

		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvMasterPort,
			Value: strconv.Itoa(int(masterPort)),
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvMasterAddr,
			Value: masterAddr,
		})
	}
	return envVars, nil
}
