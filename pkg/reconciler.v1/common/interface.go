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
// limitations under the License.

package common

import (
	"context"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcilerUtilInterface defines the abstract interface of reconciler on utility features, such like get event
// recorder or logger
type ReconcilerUtilInterface interface {
	// GetReconcilerName SHOULD be overridden if a new Reconciler is defined. The default implementation returns
	// "common-reconciler"
	GetReconcilerName() string

	// GetRecorder CAN be overridden to customize EventRecorder
	GetRecorder() record.EventRecorder

	// GetLogger CAN be overridden to customize logger
	GetLogger(job client.Object) logr.Logger

	// GetScheme CAN be overridden to customize runtime scheme
	GetScheme() *runtime.Scheme
}

// GangSchedulingInterface defines the abstract interface for gang-scheduling related actions, such like get, create or
// delete PodGroup
type GangSchedulingInterface interface {
	// OverrideForGangSchedulingInterface MUST NOT be overridden as it resets ReconcilerUtilInterface
	OverrideForGangSchedulingInterface(ui ReconcilerUtilInterface)

	// GangSchedulingEnabled CAN be overridden if definition of gang-scheduling enabling changes.
	GangSchedulingEnabled() bool

	// GetGangSchedulerName CAN be overridden to customize the name of gang scheduler. This name will be used to check
	// the value of podTemplateSpec.Spec.SchedulerName. For volcano, it is "volcano".
	GetGangSchedulerName() string

	// GetPodGroupName CAN be overridden to customize the name of PodGroup generated for the job. For example:
	// podGroupName := fmt.Sprintf("%s-podgroup", job.GetName()) or podGroupName := job.GetName()
	GetPodGroupName(job client.Object) string

	// GetPodGroupForJob SHOULD be overridden if Group, APIVersion or Kind changes for PodGroup. The PodGroup is
	// defined in different gang-scheduler as:
	// Kube-Batch:          "scheduling.incubator.k8s.io/v1alpha1/PodGroup", "scheduling.sigs.dev/v1alpha2/PodGroup"
	// Volcano:             "scheduling.volcano.sh/v1beta1/PodGroup"
	// Scheduler-Framework: "scheduling.sigs.k8s.io/v1alpha1/PodGroup"
	GetPodGroupForJob(ctx context.Context, job client.Object) (client.Object, error)

	// DeletePodGroup SHOULD be overridden if Group, APIVersion and Kind changes for PodGroup.
	DeletePodGroup(ctx context.Context, job client.Object) error

	// ReconcilePodGroup CAN be overridden if the logic to reconcile PodGroup changes.
	ReconcilePodGroup(ctx context.Context, job client.Object, runPolicy *kubeflowv1.RunPolicy,
		replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec) error

	// DecoratePodForGangScheduling SHOULD be overridden if gang scheduler demands Pods associated with PodGroup to be
	// decorated with specific requests.
	DecoratePodForGangScheduling(rtype string, podTemplate *corev1.PodTemplateSpec, job client.Object)
}

// PodInterface defines the abstract interface for Pod related actions, such like get, create or delete Pod
type PodInterface interface {
	// OverrideForPodInterface MUST NOT be overridden as it reset ReconcilerUtilInterface, GangSchedulingInterface, JobInterface
	OverrideForPodInterface(ui ReconcilerUtilInterface, gi GangSchedulingInterface, ji JobInterface)

	// GetDefaultContainerName CAN be overridden if the default container name is not "kubeflow".
	GetDefaultContainerName() string

	// GenPodName CAN be overridden to customize Pod name.
	GenPodName(jobName string, rtype string, index string) string

	// GetPodsForJob CAN be overridden to customize how to list all pods with the job.
	GetPodsForJob(ctx context.Context, job client.Object) ([]*corev1.Pod, error)

	// FilterPodsForReplicaType CAN be overridden if the linking approach between pods and replicaType changes as this
	// function filters out pods for specific replica type from all pods associated with the job.
	FilterPodsForReplicaType(pods []*corev1.Pod, replicaType string) ([]*corev1.Pod, error)

	// GetPodSlices SHOULD NOT be overridden as it generates pod slices for further pod processing.
	GetPodSlices(pods []*corev1.Pod, replicas int, logger *logrus.Entry) [][]*corev1.Pod

	// ReconcilePods CAN be overridden if the logic to reconcile all Pods for the job changes.
	ReconcilePods(
		ctx context.Context,
		job client.Object,
		jobStatus *kubeflowv1.JobStatus,
		pods []*corev1.Pod,
		rtype kubeflowv1.ReplicaType,
		spec *kubeflowv1.ReplicaSpec,
		replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec) error

	// CreateNewPod CAN be overridden to customize how to create a new pod.
	CreateNewPod(job client.Object, rt string, index string,
		spec *kubeflowv1.ReplicaSpec, masterRole bool, replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec) error

	// DeletePod CAN be overridden to customize how to delete a pod of {name} in namespace {ns}.
	DeletePod(ctx context.Context, ns string, name string) error

	// DecoratePod CAN be overridden if customization to the pod is needed. The default implementation applies nothing
	// to the pod.
	DecoratePod(rtype string, podTemplate *corev1.PodTemplateSpec, job client.Object)
}

// ServiceInterface defines the abstract interface for Pod related actions, such like get, create or delete Service
type ServiceInterface interface {
	// OverrideForServiceInterface MUST NOT be overridden as it reset ReconcilerUtilInterface, PodInterface, JobInterface
	OverrideForServiceInterface(ui ReconcilerUtilInterface, pi PodInterface, ji JobInterface)

	// GetPortsFromJob CAN be overridden to customize how to find ports defined in the ReplicasSpec.
	GetPortsFromJob(spec *kubeflowv1.ReplicaSpec) (map[string]int32, error)

	// GetServicesForJob CAN be overridden to customize how to find all services associated with this job.
	GetServicesForJob(ctx context.Context, job client.Object) ([]*corev1.Service, error)

	// FilterServicesForReplicaType CAN be overridden to customize how to filter out services for this Replica Type.
	FilterServicesForReplicaType(services []*corev1.Service, replicaType string) ([]*corev1.Service, error)

	// GetServiceSlices CAN be overridden to customize how to generate service slices.
	GetServiceSlices(services []*corev1.Service, replicas int, logger *logrus.Entry) [][]*corev1.Service

	// ReconcileServices CAN be overridden to customize how to reconcile services for this job.
	ReconcileServices(
		job client.Object,
		services []*corev1.Service,
		rtype kubeflowv1.ReplicaType,
		spec *kubeflowv1.ReplicaSpec) error

	// CreateNewService CAN be overridden to customize how to create a new service.
	CreateNewService(job client.Object, rtype kubeflowv1.ReplicaType,
		spec *kubeflowv1.ReplicaSpec, index string) error

	// DeleteService CAN be overridden to customize how to delete the service of {name} in namespace {ns}.
	DeleteService(ns string, name string, job client.Object) error

	// DecorateService CAN be overridden to customize this service right before being created
	DecorateService(rtype string, svc *corev1.Service, job client.Object)
}

// JobInterface defines the abstract interface for Pod related actions, such like get, create or delete TFJob,
// PyTorchJob or KFJob, etc.
type JobInterface interface {
	// OverrideForJobInterface MUST NOT be overridden as it reset ReconcilerUtilInterface, PodInterface, ServiceInterface, JobInterface
	OverrideForJobInterface(ui ReconcilerUtilInterface, pi PodInterface, si ServiceInterface, gi GangSchedulingInterface)

	// GenLabels CAN be overridden to customize generic label generated for Pods and Services
	GenLabels(jobName string) map[string]string

	// GetGroupNameLabelValue CAN be overridden to customize value used in labels regarding Group of job processed.
	GetGroupNameLabelValue() string

	// GetJob MUST be overridden to get jobs with specified kind
	GetJob(ctx context.Context, req ctrl.Request) (client.Object, error)

	// ExtractReplicasSpec MUST be overridden to extract ReplicasSpec from a job
	ExtractReplicasSpec(job client.Object) (map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec, error)

	// ExtractRunPolicy MUST be overridden to extract the pointer of RunPolicy from a job
	ExtractRunPolicy(job client.Object) (*kubeflowv1.RunPolicy, error)

	// ExtractJobStatus MUST be overridden to extract the pointer of JobStatus from a job
	ExtractJobStatus(job client.Object) (*kubeflowv1.JobStatus, error)

	// IsMasterRole MUST be overridden to determine whether this ReplicaType with index specified is a master role.
	// MasterRole pod will have "job-role=master" set in its label
	IsMasterRole(replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec, rtype kubeflowv1.ReplicaType, index int) bool

	// ReconcileJob CAN be overridden to customize how to reconcile a job.
	ReconcileJob(
		ctx context.Context,
		job client.Object,
		replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec,
		status *kubeflowv1.JobStatus,
		runPolicy *kubeflowv1.RunPolicy) error

	// DeleteJob CAN be overridden to customize how to delete a job.
	DeleteJob(job client.Object) error

	// UpdateJobStatus CAN be overridden to customize how to update job status without submitting to APIServer.
	UpdateJobStatus(
		job client.Object,
		replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec,
		jobStatus *kubeflowv1.JobStatus) error

	// UpdateJobStatusInAPIServer CAN be overridden to customize how to update job status directly to APIServer.
	UpdateJobStatusInAPIServer(ctx context.Context, job client.Object) error

	// CleanupResources CAN be overridden to customize how to delete all resources associated with this job.
	CleanupResources(runPolicy *kubeflowv1.RunPolicy, status kubeflowv1.JobStatus, job client.Object) error

	// CleanupJob CAN be overridden to customize how to clean up this job.
	CleanupJob(runPolicy *kubeflowv1.RunPolicy, status kubeflowv1.JobStatus, job client.Object) error

	// RecordAbnormalPods CAN be overridden to customize how to record abnormal pods
	RecordAbnormalPods(activePods []*corev1.Pod, object client.Object)

	// SetStatusForSuccessJob CAN be overridden to customize how to set status for success job
	SetStatusForSuccessJob(status *kubeflowv1.JobStatus)

	// IsFlagReplicaTypeForJobStatus CAN be overridden to customize how to determine if this ReplicaType is the
	// flag ReplicaType for the status of this kind of job
	IsFlagReplicaTypeForJobStatus(rtype string) bool

	// IsJobSucceeded CAN be overridden to customize how to determine if this job is succeeded.
	IsJobSucceeded(status kubeflowv1.JobStatus) bool

	// IsJobFailed CAN be overridden to customize how to determine if this job is failed.
	IsJobFailed(status kubeflowv1.JobStatus) bool

	// ShouldCleanUp CAN be overridden to customize how to determine if this job should be cleaned up.
	ShouldCleanUp(status kubeflowv1.JobStatus) bool

	// PastBackoffLimit CAN be overridden to customize how to determine if this job has past backoff limit.
	PastBackoffLimit(jobName string, runPolicy *kubeflowv1.RunPolicy,
		replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec, pods []*corev1.Pod) (bool, error)

	// PastActiveDeadline CAN be overridden to customize how to determine if this job has past activate deadline.
	PastActiveDeadline(runPolicy *kubeflowv1.RunPolicy, jobStatus *kubeflowv1.JobStatus) bool
}
