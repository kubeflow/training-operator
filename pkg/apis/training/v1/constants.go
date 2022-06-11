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

package v1

import commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

const (
	// MPIDefaultPortName is name of the port used to communicate between Master and Workers.
	MPIDefaultPortName = "mpi-port"
	// MPIDefaultContainerName is the name of the MPIJob container.
	MPIDefaultContainerName = "mpi"
	// MPIDefaultPort is default value of the port.
	MPIDefaultPort = 9999
	// MPIDefaultRestartPolicy is default RestartPolicy for ReplicaSpec.

	MPIDefaultRestartPolicy = commonv1.RestartPolicyNever
	MPIKind                 = "MPIJob"
	// MPIPlural is the Plural for MPIJob.
	MPIPlural = "mpijobs"
	// MPISingular is the singular for MPIJob.
	MPISingular = "mpijob"
	// MPIFrameworkName is the name of the ML Framework
	MPIFrameworkName = "mpi"
)

const (
	// XGBoostDefaultPortName is name of the port used to communicate between Master and Workers.
	XGBoostDefaultPortName = "xgboostjob-port"
	// XGBoostDefaultContainerName is the name of the XGBoostJob container.
	XGBoostDefaultContainerName = "xgboost"
	// XGBoostDefaultPort is default value of the port.
	XGBoostDefaultPort = 9999
	// XGBoostDefaultRestartPolicy is default RestartPolicy for XGBReplicaSpecs.
	XGBoostDefaultRestartPolicy = commonv1.RestartPolicyNever
	// XGBoostKind is the kind name.
	XGBoostKind = "XGBoostJob"
	// Plural is the Plural for XGBoostJob.
	XGBoostPlural = "xgboostjobs"
	// XGBoostSingular is the singular for XGBoostJob.
	XGBoostSingular = "xgboostjob"
	// XGBoostFrameworkName is the name of the ML Framework
	XGBoostFrameworkName = "xgboost"
)

const (
	// MXDefaultPortName is name of the port used to communicate between scheduler and
	// servers & workers.
	MXDefaultPortName = "mxjob-port"
	// MXDefaultContainerName is the name of the MXJob container.
	MXDefaultContainerName = "mxnet"
	// MXDefaultPort is default value of the port.
	MXDefaultPort = 9091
	// MXDefaultRestartPolicy is default RestartPolicy for MXReplicaSpec.
	MXDefaultRestartPolicy = commonv1.RestartPolicyNever
	// MXKind is the kind name.
	MXKind = "MXJob"
	// MXPlural is the Plural for mxJob.
	MXPlural = "mxjobs"
	// MXSingular is the singular for mxJob.
	MXSingular = "mxjob"
	// MXFrameworkName is the name of the ML Framework
	MXFrameworkName = "mxnet"
)

const (
	// PyTorchDefaultPortName is name of the port used to communicate between Master and
	// workers.
	PyTorchDefaultPortName = "pytorchjob-port"
	// PyTorchDefaultContainerName is the name of the PyTorchJob container.
	PyTorchDefaultContainerName = "pytorch"
	// PyTorchDefaultPort is default value of the port.
	PyTorchDefaultPort = 23456
	// PyTorchDefaultRestartPolicy is default RestartPolicy for PyTorchReplicaSpec.
	PyTorchDefaultRestartPolicy = commonv1.RestartPolicyOnFailure
	// PyTorchKind is the kind name.
	PyTorchKind = "PyTorchJob"
	// PyTorchPlural is the Plural for pytorchJob.
	PyTorchPlural = "pytorchjobs"
	// PyTorchSingular is the singular for pytorchJob.
	PyTorchSingular = "pytorchjob"
	// PyTorchFrameworkName is the name of the ML Framework
	PyTorchFrameworkName = "pytorch"
)

const (
	// TFDefaultPortName is name of the port used to communicate between PS and
	// workers.
	TFDefaultPortName = "tfjob-port"
	// TFDefaultContainerName is the name of the TFJob container.
	TFDefaultContainerName = "tensorflow"
	// TFDefaultPort is default value of the port.
	TFDefaultPort = 2222
	// TFDefaultRestartPolicy is default RestartPolicy for TFReplicaSpec.
	TFDefaultRestartPolicy = commonv1.RestartPolicyNever
	// TFKind is the kind name.
	TFKind = "TFJob"
	// TFPlural is the Plural for TFJob.
	TFPlural = "tfjobs"
	// TFSingular is the singular for TFJob.
	TFSingular = "tfjob"
	// TFFrameworkName is the name of the ML Framework
	TFFrameworkName = "tensorflow"
)
