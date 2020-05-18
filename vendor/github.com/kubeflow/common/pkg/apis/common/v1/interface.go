package v1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ControllerInterface defines the Interface to be implemented by custom operators. e.g. tf-operator needs to implement this interface
type ControllerInterface interface {
	// Returns the Controller name
	ControllerName() string

	// Returns the GroupVersionKind of the API
	GetAPIGroupVersionKind() schema.GroupVersionKind

	// Returns the GroupVersion of the API
	GetAPIGroupVersion() schema.GroupVersion

	// Returns the Group Name(value) in the labels of the job
	GetGroupNameLabelValue() string

	// Returns the Job from Informer Cache
	GetJobFromInformerCache(namespace, name string) (metav1.Object, error)

	// Returns the Job from API server
	GetJobFromAPIClient(namespace, name string) (metav1.Object, error)

	// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
	// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
	GetPodsForJob(job interface{}) ([]*v1.Pod, error)

	// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
	// i.e. all services created by the job will come with label "job-name" = <this_job_name>
	GetServicesForJob(job interface{}) ([]*v1.Service, error)

	// DeleteJob deletes the job
	DeleteJob(job interface{}) error

	// UpdateJobStatus updates the job status and job conditions
	UpdateJobStatus(job interface{}, replicas map[ReplicaType]*ReplicaSpec, jobStatus *JobStatus) error

	// UpdateJobStatusInApiServer updates the job status in API server
	UpdateJobStatusInApiServer(job interface{}, jobStatus *JobStatus) error

	// SetClusterSpec sets the cluster spec for the pod
	SetClusterSpec(job interface{}, podTemplate *v1.PodTemplateSpec, rtype, index string) error

	// Returns the default container name in pod
	GetDefaultContainerName() string

	// Get the default container port name
	GetDefaultContainerPortName() string

	// Returns if this replica type with index specified is a master role.
	// MasterRole pod will have "job-role=master" set in its label
	IsMasterRole(replicas map[ReplicaType]*ReplicaSpec, rtype ReplicaType, index int) bool

	// ReconcileJobs checks and updates replicas for each given ReplicaSpec of a job.
	// Common implementation will be provided and User can still override this to implement their own reconcile logic
	ReconcileJobs(job interface{}, replicas map[ReplicaType]*ReplicaSpec, jobStatus JobStatus, runPolicy *RunPolicy) error

	// ReconcilePods checks and updates pods for each given ReplicaSpec.
	// It will requeue the job in case of an error while creating/deleting pods.
	// Common implementation will be provided and User can still override this to implement their own reconcile logic
	ReconcilePods(job interface{}, jobStatus *JobStatus, pods []*v1.Pod, rtype ReplicaType, spec *ReplicaSpec,
		replicas map[ReplicaType]*ReplicaSpec) error

	// ReconcileServices checks and updates services for each given ReplicaSpec.
	// It will requeue the job in case of an error while creating/deleting services.
	// Common implementation will be provided and User can still override this to implement their own reconcile logic
	ReconcileServices(job metav1.Object, services []*v1.Service, rtype ReplicaType, spec *ReplicaSpec) error
}
