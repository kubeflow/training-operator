package constants

const (

	// JobSetKind is the Kind name for the JobSet.
	JobSetKind string = "JobSet"

	// PodGroupKind is the Kind name for the PodGroup.
	PodGroupKind string = "PodGroup"

	// JobTrainerNode is the Job name for the trainer node.
	JobTrainerNode string = "trainer-node"

	// ContainerTrainer is the container name for the trainer.
	ContainerTrainer string = "trainer"

	// JobInitializer is the Job name for the initializer.
	JobInitializer string = "initializer"

	// ContainerModelInitializer is the container name for the model initializer.
	ContainerModelInitializer string = "model-initializer"

	// ContainerDatasetInitializer is the container name for the dataset initializer.
	ContainerDatasetInitializer string = "dataset-initializer"

	// Distributed envs for torchrun.
	// Ref: https://github.com/pytorch/pytorch/blob/3a0d0885171376ed610c8175a19ba40411fc6f3f/torch/distributed/argparse_util.py#L45
	// Number of training nodes.
	TorchEnvNumNodes string = "PET_NNODES"

	// Number of procs (e.g. GPUs) per node.
	TorchEnvNumProcPerNode string = "PET_NPROC_PER_NODE"
)
