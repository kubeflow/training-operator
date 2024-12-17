package pytorch

import (
	"strconv"
	"sync"

	corev1 "k8s.io/api/core/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

var (
	masterGenerator EnvVarGenerator
	onceMaster      sync.Once
	EnvMasterPort   = "MASTER_PORT"
	EnvMasterAddr   = "MASTER_ADDR"

	PETMasterPort = "PET_MASTER_PORT"
	PETMasterAddr = "PET_MASTER_ADDR"
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
	job *kubeflowv1.PyTorchJob) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar
	if job.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeMaster] != nil {
		masterPort, err := getPortFromPyTorchJob(job, kubeflowv1.PyTorchJobReplicaTypeMaster)
		if err != nil {
			return nil, err
		}

		masterAddr := replicaName(job.Name, kubeflowv1.PyTorchJobReplicaTypeMaster, 0)

		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvMasterPort,
			Value: strconv.Itoa(int(masterPort)),
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  PETMasterPort,
			Value: strconv.Itoa(int(masterPort)),
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvMasterAddr,
			Value: masterAddr,
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  PETMasterAddr,
			Value: masterAddr,
		})
	}
	return envVars, nil
}
