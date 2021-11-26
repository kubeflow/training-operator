package pytorch

import (
	"strconv"
	"strings"
	"sync"

	pytorchv1 "github.com/kubeflow/training-operator/pkg/apis/pytorch/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	masterGenerator EnvVarGenerator
	once            sync.Once
	EnvMasterPort   = "MASTER_PORT"
	EnvMasterAddr   = "MASTER_ADDR"
)

type MasterEnvVarGenerator struct {
}

func GetMasterEnvVarGenerator() EnvVarGenerator {
	once.Do(func() {
		masterGenerator = &MasterEnvVarGenerator{}
	})
	return masterGenerator
}

func (e MasterEnvVarGenerator) Generate(
	job *pytorchv1.PyTorchJob) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}
	if job.Spec.PyTorchReplicaSpecs[pytorchv1.PyTorchReplicaTypeMaster] != nil {
		masterPort, err := getPortFromPyTorchJob(job, pytorchv1.PyTorchReplicaTypeMaster)
		if err != nil {
			return nil, err
		}

		masterAddr := genGeneralName(job.Name, strings.ToLower(string(pytorchv1.PyTorchReplicaTypeMaster)), strconv.Itoa(0))

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
