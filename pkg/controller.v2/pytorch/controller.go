// Copyright 2018 The Kubeflow Authors
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

// Package controller provides a Kubernetes controller for a PyTorchJob resource.
package pytorch

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeflow/tf-operator/cmd/pytorch-operator/app/options"
	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
	jobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	jobscheme "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	jobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	jobinformersv1alpha2 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/pytorch/v1alpha2"
	joblisters "github.com/kubeflow/tf-operator/pkg/client/listers/pytorch/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/controller.v2/jobcontroller"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	controllerName = "pytorch-operator"

	// labels for pods and servers.
	replicaTypeLabel    = "pytorch-replica-type"
	replicaIndexLabel   = "pytorch-replica-index"
	labelGroupName      = "group_name"
	labelPyTorchJobName = "pytorch_job_name"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// DefaultPyTorchControllerConfiguration is the suggested operator configuration for production.
	DefaultPyTorchControllerConfiguration = jobcontroller.JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: metav1.Duration{Duration: 15 * time.Second},
		EnableGangScheduling:     false,
	}
)

// PyTorchController is the type for PyTorchJob Controller, which manages
// the lifecycle of PyTorchJobs.
type PyTorchController struct {
	jobcontroller.JobController

	// jobClientSet is a clientset for CRD PyTorchJob.
	jobClientSet jobclientset.Interface

	// To allow injection of sync functions for testing.
	syncHandler func(string) (bool, error)

	// To allow injection of updateStatus for testing.
	updateStatusHandler func(job *v1alpha2.PyTorchJob) error

	// To allow injection of deletePyTorchJob for testing.
	deletePyTorchJobHandler func(job *v1alpha2.PyTorchJob) error

	// jobInformer is a temporary field for unstructured informer support.
	jobInformer cache.SharedIndexInformer

	// Listers for PyTorchJob, Pod and Service
	// jobLister can list/get jobs from the shared informer's store.
	jobLister joblisters.PyTorchJobLister

	// jobInformerSynced returns true if the job store has been synced at least once.
	jobInformerSynced cache.InformerSynced
}

// NewPyTorchController returns a new PyTorchJob controller.
func NewPyTorchController(
	// This variable is for unstructured informer.
	jobInformer jobinformersv1alpha2.PyTorchJobInformer,
	kubeClientSet kubeclientset.Interface,
	jobClientSet jobclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	// This field is not used now but we keep it since it will be used
	// after we support CRD validation.
	jobInformerFactory jobinformers.SharedInformerFactory,
	option options.ServerOption) *PyTorchController {

	jobscheme.AddToScheme(scheme.Scheme)

	log.Info("Creating PyTorchJob controller")
	// Create new PyTorchController.
	pc := &PyTorchController{
		jobClientSet: jobClientSet,
	}

	// Create base controller
	log.Info("Creating Job controller")
	jc := jobcontroller.NewJobController(pc, metav1.Duration{Duration: 15 * time.Second},
		option.EnableGangScheduling, kubeClientSet, kubeInformerFactory, v1alpha2.Plural)
	pc.JobController = jc
	// Set sync handler.
	pc.syncHandler = pc.syncPyTorchJob
	pc.updateStatusHandler = pc.updatePyTorchJobStatus
	// set delete handler.
	pc.deletePyTorchJobHandler = pc.deletePyTorchJob
	// Set up an event handler for when job resources change.
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    pc.addPyTorchJob,
		UpdateFunc: pc.updatePyTorchJob,
		// This will enter the sync loop and no-op,
		// because the job has been deleted from the store.
		DeleteFunc: pc.enqueuePyTorchJob,
	})

	pc.jobInformer = jobInformer.Informer()
	pc.jobLister = jobInformer.Lister()
	pc.jobInformerSynced = jobInformer.Informer().HasSynced

	// Create pod informer.
	podInformer := kubeInformerFactory.Core().V1().Pods()

	// Set up an event handler for when pod resources change
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    jc.AddPod,
		UpdateFunc: jc.UpdatePod,
		DeleteFunc: jc.DeletePod,
	})

	pc.PodLister = podInformer.Lister()
	pc.PodInformerSynced = podInformer.Informer().HasSynced

	// Create service informer.
	serviceInformer := kubeInformerFactory.Core().V1().Services()

	// Set up an event handler for when service resources change.
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    jc.AddService,
		UpdateFunc: jc.UpdateService,
		DeleteFunc: jc.DeleteService,
	})

	pc.ServiceLister = serviceInformer.Lister()
	pc.ServiceInformerSynced = serviceInformer.Informer().HasSynced

	return pc
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (pc *PyTorchController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer pc.WorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches.
	log.Info("Starting PyTorchJob controller")

	// Wait for the caches to be synced before starting workers.
	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, pc.jobInformerSynced); !ok {
		return fmt.Errorf("failed to wait for job caches to sync")
	}

	if ok := cache.WaitForCacheSync(stopCh, pc.PodInformerSynced); !ok {
		return fmt.Errorf("failed to wait for pod caches to sync")
	}

	if ok := cache.WaitForCacheSync(stopCh, pc.ServiceInformerSynced); !ok {
		return fmt.Errorf("failed to wait for service caches to sync")
	}

	log.Infof("Starting %v workers", threadiness)
	// Launch workers to process PyTorchJob resources.
	for i := 0; i < threadiness; i++ {
		go wait.Until(pc.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (pc *PyTorchController) runWorker() {
	for pc.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (pc *PyTorchController) processNextWorkItem() bool {
	key, quit := pc.WorkQueue.Get()
	if quit {
		return false
	}
	defer pc.WorkQueue.Done(key)

	logger := pylogger.LoggerForKey(key.(string))

	pytorchJob, err := pc.getPyTorchJobFromKey(key.(string))
	if err != nil {
		if err == errNotExists {
			logger.Infof("PyTorchJob has been deleted: %v", key)
			return true
		}

		// Log the failure to conditions.
		logger.Errorf("Failed to get PyTorchJob from key %s: %v", key, err)
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to PyTorchJob object: %v", err)
			pylogger.LoggerForJob(pytorchJob).Warn(errMsg)
			pc.Recorder.Event(pytorchJob, v1.EventTypeWarning, failedMarshalPyTorchJobReason, errMsg)
		}

		return true
	}

	// Sync PyTorchJob to mapch the actual state to this desired state.
	forget, err := pc.syncHandler(key.(string))
	if err == nil {
		if forget {
			pc.WorkQueue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing job: %v", err))
	pc.WorkQueue.AddRateLimited(key)

	return true
}

func (pc *PyTorchController) enqueuePyTorchJob(job interface{}) {
	key, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for job object %#v: %v", job, err))
		return
	}

	// TODO: we may need add backoff here
	pc.WorkQueue.Add(key)
}

// syncPyTorchJob syncs the job with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods/services created or deleted.
// This function is not meant to be invoked concurrently with the same key.
func (pc *PyTorchController) syncPyTorchJob(key string) (bool, error) {
	startTime := time.Now()
	logger := pylogger.LoggerForKey(key)
	defer func() {
		logger.Infof("Finished syncing job %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	if len(namespace) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid job key %q: either namespace or name is missing", key)
	}

	sharedJob, err := pc.getPyTorchJobFromName(namespace, name)
	if err != nil {
		if err == errNotExists {
			logger.Infof("PyTorchJob has been deleted: %v", key)
			// jm.expectations.DeleteExpectations(key)
			return true, nil
		}
		return false, err
	}

	job := sharedJob.DeepCopy()
	jobNeedsSync := pc.satisfiedExpectations(job)

	if pc.Config.EnableGangScheduling {
		_, err := pc.SyncPdb(job)
		if err != nil {
			logger.Warnf("Sync pdb %v: %v", job.Name, err)
		}
	}

	// Set default for the new job.
	scheme.Scheme.Default(job)

	var reconcilePyTorchJobsErr error
	if jobNeedsSync && job.DeletionTimestamp == nil {
		reconcilePyTorchJobsErr = pc.reconcilePyTorchJobs(job)
	}

	if reconcilePyTorchJobsErr != nil {
		return false, reconcilePyTorchJobsErr
	}

	return true, err
}

func (pc *PyTorchController) GetTotalReplicas(obj metav1.Object) int32 {
	job := obj.(*v1alpha2.PyTorchJob)
	jobReplicas := int32(0)
	for _, r := range job.Spec.PyTorchReplicaSpecs {
		jobReplicas += *r.Replicas
	}
	return jobReplicas
}

// reconcilePyTorchJobs checks and updates replicas for each given PyTorchReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (pc *PyTorchController) reconcilePyTorchJobs(job *v1alpha2.PyTorchJob) error {
	logger := pylogger.LoggerForJob(job)
	logger.Infof("Reconcile PyTorchJobs %s", job.Name)

	pods, err := pc.GetPodsForJob(job)

	if err != nil {
		logger.Warnf("getPodsForPyTorchJob error %v", err)
		return err
	}

	services, err := pc.GetServicesForJob(job)

	if err != nil {
		logger.Warnf("getServicesForPyTorchJob error %v", err)
		return err
	}

	// If the PyTorchJob is terminated, delete all pods and services.
	if isSucceeded(job.Status) || isFailed(job.Status) {
		if err := pc.deletePodsAndServices(job, pods); err != nil {
			return err
		}

		if err := pc.cleanupPyTorchJob(job); err != nil {
			return err
		}

		if pc.Config.EnableGangScheduling {
			pc.Recorder.Event(job, v1.EventTypeNormal, "JobTerminated", "Job is terminated, deleting pdb")
			if err := pc.DeletePdb(job); err != nil {
				pc.Recorder.Eventf(job, v1.EventTypeWarning, "FailedDeletePdb", "Error deleting: %v", err)
				return err
			} else {
				pc.Recorder.Eventf(job, v1.EventTypeNormal, "SuccessfulDeletePdb", "Deleted pdb: %v", job.Name)

			}
		}

		// Initialize the status.
		initializePyTorchReplicaStatuses(job, v1alpha2.PyTorchReplicaTypeMaster)
		initializePyTorchReplicaStatuses(job, v1alpha2.PyTorchReplicaTypeWorker)
		return pc.updateStatusHandler(job)
	}

	// Save the current state of the replicas
	replicasStatus := make(map[string]v1.PodPhase)

	// Diff current active pods/services with replicas.
	for rtype, spec := range job.Spec.PyTorchReplicaSpecs {
		err = pc.reconcilePods(job, pods, rtype, spec, replicasStatus)
		if err != nil {
			logger.Warnf("reconcilePods error %v", err)
			return err
		}

		err = pc.reconcileServices(job, services, rtype, spec)

		if err != nil {
			logger.Warnf("reconcileServices error %v", err)
			return err
		}
	}

	// TODO(CPH): Add check here, no need to update the job if the status hasn't changed since last time.
	return pc.updateStatusHandler(job)
}

// satisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (pc *PyTorchController) satisfiedExpectations(job *v1alpha2.PyTorchJob) bool {
	satisfied := false
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for job object %#v: %v", job, err))
		return false
	}

	for rtype := range job.Spec.PyTorchReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := jobcontroller.GenExpectationPodsKey(jobKey, string(rtype))
		satisfied = satisfied || pc.Expectations.SatisfiedExpectations(expectationPodsKey)

		// Check the expectations of the services.
		expectationServicesKey := jobcontroller.GenExpectationServicesKey(jobKey, string(rtype))
		satisfied = satisfied || pc.Expectations.SatisfiedExpectations(expectationServicesKey)
	}

	return satisfied
}

func (pc *PyTorchController) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	return pc.getPyTorchJobFromName(namespace, name)
}

func (pc *PyTorchController) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	return pc.jobClientSet.Pytorch().PyTorchJobs(namespace).Get(name, metav1.GetOptions{})
}

func (pc *PyTorchController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return v1alpha2.SchemeGroupVersionKind
}

func (pc *PyTorchController) GetAPIGroupVersion() schema.GroupVersion {
	return v1alpha2.SchemeGroupVersion
}

func (pc *PyTorchController) GetGroupNameLabelKey() string {
	return labelGroupName
}

func (pc *PyTorchController) GetJobNameLabelKey() string {
	return labelPyTorchJobName
}

func (pc *PyTorchController) GetGroupNameLabelValue() string {
	return v1alpha2.GroupName
}

func (pc *PyTorchController) GetReplicaTypeLabelKey() string {
	return replicaTypeLabel
}

func (pc *PyTorchController) GetReplicaIndexLabelKey() string {
	return replicaIndexLabel
}

func (pc *PyTorchController) ControllerName() string {
	return controllerName
}
