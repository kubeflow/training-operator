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
	"fmt"
	"strconv"
	"sync"

	corev1 "k8s.io/api/core/v1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

const (
	// Rendezvous related arguments

	// EnvRDZVBackend is the environment variable name for the rdzv backend.
	EnvRDZVBackend = "PET_RDZV_BACKEND"
	// EnvRDZVID is the environment variable name for the rdzv id.
	EnvRDZVID = "PET_RDZV_ID"
	// ENVRDZVConf is the environment variable name for the rdzv conf.
	EnvRDZVConf = "PET_RDZV_CONF"
	// EnvRDZVEndpoint is the environment variable name for the rdzv endpoint.
	EnvRDZVEndpoint = "PET_RDZV_ENDPOINT"
	// EnvRDZVStandalone is the environment variable name for the standalone mode.
	EnvStandalone = "PET_STANDALONE"

	// User-code launch related arguments.

	// EnvMaxRestarts is the environment variable name for the maximum number of worker group restarts before failing.
	EnvMaxRestarts = "PET_MAX_RESTARTS"
	// EnvMonitorInterval is the environment variable name for the interval, in seconds, to monitor the state of workers.
	EnvMonitorInterval = "PET_MONITOR_INTERVAL"
	// EnvStartMethod is the environment variable name for the multiprocessing start method to use when creating workers, which could be fork, spawn and forkserver.
	EnvStartMethod = "PET_START_METHOD"

	// Worker/node size related arguments.

	// EnvNProcPerNode is the environment variable name for the number of processes per node.
	EnvNProcPerNode = "PET_NPROC_PER_NODE"
	// EnvNNodes is the environment variable name for the number of nodes.
	EnvNNodes = "PET_NNODES"
)

var (
	elasticGenerator EnvVarGenerator
	onceElastic      sync.Once
)

// ElasticEnvVarGenerator is the environment variable generator for Elastic related arguments.
type ElasticEnvVarGenerator struct{}

func GetElasticEnvVarGenerator() EnvVarGenerator {
	onceElastic.Do(func() {
		elasticGenerator = &ElasticEnvVarGenerator{}
	})
	return elasticGenerator
}

func (e ElasticEnvVarGenerator) Generate(
	job *kubeflowv1.PyTorchJob) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}

	elasticPolicy := job.Spec.ElasticPolicy
	if elasticPolicy == nil {
		// Return empty env vars.
		return nil, nil
	}

	// Generate RDZV_ENDPOINT.
	if envVar, err := e.generateEnvRDZVEndpoint(job); err != nil {
		return nil, err
	} else {
		envVars = append(envVars, *envVar)
	}
	// Generate RDZV_BACKEND.
	envVars = append(envVars, e.generateEnvBackend(elasticPolicy))
	// Generate NNODES.
	if envVar, err := e.generateEnvNNodes(job); err != nil {
		return nil, err
	} else {
		envVars = append(envVars, *envVar)
	}

	if elasticPolicy.MaxRestarts != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvMaxRestarts,
			Value: strconv.Itoa(int(*elasticPolicy.MaxRestarts)),
		})
	}
	if elasticPolicy.NProcPerNode != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvNProcPerNode,
			Value: strconv.Itoa(int(*elasticPolicy.NProcPerNode)),
		})
	}
	if elasticPolicy.RDZVID != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvRDZVID,
			Value: *elasticPolicy.RDZVID,
		})
	}
	if envVar := e.generateEnvRDZVConf(elasticPolicy); envVar != nil {
		envVars = append(envVars, *envVar)
	}
	if elasticPolicy.Standalone != nil && *elasticPolicy.Standalone {
		envVars = append(envVars, corev1.EnvVar{
			Name:  EnvStandalone,
			Value: "",
		})
	}

	return envVars, nil
}

func (e ElasticEnvVarGenerator) generateEnvNNodes(job *kubeflowv1.PyTorchJob) (*corev1.EnvVar, error) {
	// Return worker.replicas if there is no max and min replicas specified.
	if job.Spec.ElasticPolicy.MinReplicas == nil &&
		job.Spec.ElasticPolicy.MaxReplicas == nil {
		if job.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeWorker] == nil {
			return nil, fmt.Errorf("cannot find the worker spec")
		}
		return &corev1.EnvVar{
			Name: EnvNNodes,
			Value: strconv.Itoa(
				int(*job.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeWorker].
					Replicas)),
		}, nil
	}

	return &corev1.EnvVar{
		Name: EnvNNodes,
		Value: fmt.Sprintf("%d:%d",
			*job.Spec.ElasticPolicy.MinReplicas, *job.Spec.ElasticPolicy.MaxReplicas),
	}, nil
}

func (e ElasticEnvVarGenerator) generateEnvRDZVEndpoint(job *kubeflowv1.PyTorchJob) (*corev1.EnvVar, error) {
	var err error
	host := ""
	if job.Spec.ElasticPolicy.RDZVHost == nil {
		host = fmt.Sprintf("%s-worker-0", job.Name)
	} else {
		host = *job.Spec.ElasticPolicy.RDZVHost
	}

	var port int32
	if job.Spec.ElasticPolicy.RDZVPort == nil {
		// Generate RDZV_Endpoint.
		port, err = getPortFromPyTorchJob(job, kubeflowv1.PyTorchJobReplicaTypeWorker)
		if err != nil {
			return nil, err
		}
	} else {
		port = *job.Spec.ElasticPolicy.RDZVPort
	}
	return &corev1.EnvVar{
		Name:  EnvRDZVEndpoint,
		Value: fmt.Sprintf("%s:%d", host, port),
	}, nil
}

func (e ElasticEnvVarGenerator) generateEnvRDZVConf(elasticPolicy *kubeflowv1.ElasticPolicy) *corev1.EnvVar {
	if elasticPolicy.RDZVConf == nil {
		return nil
	}
	val := ""
	for _, conf := range elasticPolicy.RDZVConf {
		val += fmt.Sprintf("%s=%s,", conf.Key, conf.Value)
	}
	return &corev1.EnvVar{
		Name: EnvRDZVConf,
		// Remove the last comma.
		Value: val[:len(val)-1],
	}
}

func (e ElasticEnvVarGenerator) generateEnvBackend(elasticPolicy *kubeflowv1.ElasticPolicy) corev1.EnvVar {
	if elasticPolicy.RDZVBackend != nil {
		return corev1.EnvVar{
			Name:  EnvRDZVBackend,
			Value: string(*elasticPolicy.RDZVBackend),
		}
	}
	return corev1.EnvVar{
		Name:  EnvRDZVBackend,
		Value: string(kubeflowv1.BackendC10D),
	}
}
