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

// Package controller provides a Kubernetes controller for a TFJob resource.
package tensorflow

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

	"github.com/kubeflow/tf-operator/cmd/tf-operator.v1beta1/app/options"
	tfv1beta1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta1"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobscheme "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	tfjobinformersv1beta1 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/kubeflow/v1beta1"
	tfjoblisters "github.com/kubeflow/tf-operator/pkg/client/listers/kubeflow/v1beta1"
	"github.com/kubeflow/tf-operator/pkg/common/jobcontroller"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	controllerName = "tf-operator"

	// labels for pods and servers.
	tfReplicaTypeLabel  = "tf-replica-type"
	tfReplicaIndexLabel = "tf-replica-index"
	labelGroupName      = "group_name"
	labelTFJobName      = "tf_job_name"
	labelTFJobRole      = "tf_job_role"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// DefaultTFControllerConfiguration is the suggested tf-operator configuration for production.
	DefaultTFControllerConfiguration = jobcontroller.JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: metav1.Duration{Duration: 15 * time.Second},
		EnableGangScheduling:     false,
	}
)

// TFController is the type for TFJob Controller, which manages
// the lifecycle of TFJobs.
type TFController struct {
	jobcontroller.JobController

	// tfJobClientSet is a clientset for CRD TFJob.
	tfJobClientSet tfjobclientset.Interface

	// To allow injection of sync functions for testing.
	syncHandler func(string) (bool, error)

	// To allow injection of updateStatus for testing.
	updateStatusHandler func(tfjob *tfv1beta1.TFJob) error

	// To allow injection of deleteTFJob for testing.
	deleteTFJobHandler func(tfjob *tfv1beta1.TFJob) error

	// tfJobInformer is a temporary field for unstructured informer support.
	tfJobInformer cache.SharedIndexInformer

	// Listers for TFJob, Pod and Service
	// tfJobLister can list/get tfjobs from the shared informer's store.
	tfJobLister tfjoblisters.TFJobLister

	// tfJobInformerSynced returns true if the tfjob store has been synced at least once.
	tfJobInformerSynced cache.InformerSynced
}

// NewTFController returns a new TFJob controller.
func NewTFController(
	// This variable is for unstructured informer.
	tfJobInformer tfjobinformersv1beta1.TFJobInformer,
	kubeClientSet kubeclientset.Interface,
	tfJobClientSet tfjobclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	// This field is not used now but we keep it since it will be used
	// after we support CRD validation.
	tfJobInformerFactory tfjobinformers.SharedInformerFactory,
	option options.ServerOption) *TFController {

	tfjobscheme.AddToScheme(scheme.Scheme)

	log.Info("Creating TFJob controller")
	// Create new TFController.
	tc := &TFController{
		tfJobClientSet: tfJobClientSet,
	}

	// Create base controller
	log.Info("Creating Job controller")
	jc := jobcontroller.NewJobController(tc, metav1.Duration{Duration: 15 * time.Second},
		option.EnableGangScheduling, kubeClientSet, kubeInformerFactory, tfv1beta1.Plural)
	tc.JobController = jc
	// Set sync handler.
	tc.syncHandler = tc.syncTFJob
	tc.updateStatusHandler = tc.updateTFJobStatus
	// set delete handler.
	tc.deleteTFJobHandler = tc.deleteTFJob
	// Set up an event handler for when tfjob resources change.
	tfJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    tc.addTFJob,
		UpdateFunc: tc.updateTFJob,
		// This will enter the sync loop and no-op,
		// because the tfjob has been deleted from the store.
		DeleteFunc: tc.enqueueTFJob,
	})

	tc.tfJobInformer = tfJobInformer.Informer()
	tc.tfJobLister = tfJobInformer.Lister()
	tc.tfJobInformerSynced = tfJobInformer.Informer().HasSynced

	// Create pod informer.
	podInformer := kubeInformerFactory.Core().V1().Pods()

	// Set up an event handler for when pod resources change
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    jc.AddPod,
		UpdateFunc: jc.UpdatePod,
		DeleteFunc: jc.DeletePod,
	})

	tc.PodLister = podInformer.Lister()
	tc.PodInformerSynced = podInformer.Informer().HasSynced

	// Create service informer.
	serviceInformer := kubeInformerFactory.Core().V1().Services()

	// Set up an event handler for when service resources change.
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    jc.AddService,
		UpdateFunc: jc.UpdateService,
		DeleteFunc: jc.DeleteService,
	})

	tc.ServiceLister = serviceInformer.Lister()
	tc.ServiceInformerSynced = serviceInformer.Informer().HasSynced

	return tc
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (tc *TFController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer tc.WorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches.
	log.Info("Starting TFJob controller")

	// Wait for the caches to be synced before starting workers.
	log.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, tc.tfJobInformerSynced,
		tc.PodInformerSynced, tc.ServiceInformerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	log.Infof("Starting %v workers", threadiness)
	// Launch workers to process TFJob resources.
	for i := 0; i < threadiness; i++ {
		go wait.Until(tc.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (tc *TFController) runWorker() {
	for tc.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (tc *TFController) processNextWorkItem() bool {
	obj, quit := tc.WorkQueue.Get()
	if quit {
		return false
	}
	defer tc.WorkQueue.Done(obj)

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		// As the item in the workqueue is actually invalid, we call
		// Forget here else we'd go into a loop of attempting to
		// process a work item that is invalid.
		tc.WorkQueue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		return true
	}
	logger := tflogger.LoggerForKey(key)

	tfJob, err := tc.getTFJobFromKey(key)
	if err != nil {
		if err == errNotExists {
			logger.Infof("TFJob has been deleted: %v", key)
			return true
		}

		// Log the failure to conditions.
		logger.Errorf("Failed to get TFJob from key %s: %v", key, err)
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to TFJob object: %v", err)
			tflogger.LoggerForJob(tfJob).Warn(errMsg)
			tc.Recorder.Event(tfJob, v1.EventTypeWarning, failedMarshalTFJobReason, errMsg)
		}

		return true
	}

	// Sync TFJob to match the actual state to this desired state.
	forget, err := tc.syncHandler(key)
	if err == nil {
		if forget {
			tc.WorkQueue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("error syncing tfjob: %v", err))
	tc.WorkQueue.AddRateLimited(key)

	return true
}

func (tc *TFController) enqueueTFJob(tfjob interface{}) {
	key, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfjob, err))
		return
	}

	// TODO: we may need add backoff here
	tc.WorkQueue.Add(key)
}

// syncTFJob syncs the tfjob with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods/services created or deleted.
// This function is not meant to be invoked concurrently with the same key.
func (tc *TFController) syncTFJob(key string) (bool, error) {
	startTime := time.Now()
	logger := tflogger.LoggerForKey(key)
	defer func() {
		logger.Infof("Finished syncing tfjob %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	if len(namespace) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid tfjob key %q: either namespace or name is missing", key)
	}

	sharedTFJob, err := tc.getTFJobFromName(namespace, name)
	if err != nil {
		if err == errNotExists {
			logger.Infof("TFJob has been deleted: %v", key)
			// jm.expectations.DeleteExpectations(key)
			return true, nil
		}
		return false, err
	}

	tfjob := sharedTFJob.DeepCopy()
	tfjobNeedsSync := tc.satisfiedExpectations(tfjob)

	if tc.Config.EnableGangScheduling {
		minAvailableReplicas := getTotalReplicas(tfjob)
		_, err := tc.SyncPdb(tfjob, minAvailableReplicas)
		if err != nil {
			logger.Warnf("Sync pdb %v: %v", tfjob.Name, err)
		}
	}

	// Set default for the new tfjob.
	scheme.Scheme.Default(tfjob)

	var reconcileTFJobsErr error
	if tfjobNeedsSync && tfjob.DeletionTimestamp == nil {
		reconcileTFJobsErr = tc.reconcileTFJobs(tfjob)
	}

	if reconcileTFJobsErr != nil {
		return false, reconcileTFJobsErr
	}

	return true, err
}

func getTotalReplicas(tfjob *tfv1beta1.TFJob) int32 {
	tfjobReplicas := int32(0)
	for _, r := range tfjob.Spec.TFReplicaSpecs {
		tfjobReplicas += *r.Replicas
	}
	return tfjobReplicas
}

// reconcileTFJobs checks and updates replicas for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting pods/services.
func (tc *TFController) reconcileTFJobs(tfjob *tfv1beta1.TFJob) error {
	logger := tflogger.LoggerForJob(tfjob)
	logger.Infof("Reconcile TFJobs %s", tfjob.Name)

	pods, err := tc.GetPodsForJob(tfjob)

	if err != nil {
		logger.Warnf("getPodsForTFJob error %v", err)
		return err
	}

	services, err := tc.GetServicesForJob(tfjob)

	if err != nil {
		logger.Warnf("getServicesForTFJob error %v", err)
		return err
	}

	// If the TFJob is terminated, delete all pods and services.
	if isSucceeded(tfjob.Status) || isFailed(tfjob.Status) {
		if err := tc.deletePodsAndServices(tfjob, pods); err != nil {
			return err
		}

		if err := tc.cleanupTFJob(tfjob); err != nil {
			return err
		}

		if tc.Config.EnableGangScheduling {
			tc.Recorder.Event(tfjob, v1.EventTypeNormal, "JobTerminated", "Job is terminated, deleting pdb")
			if err := tc.DeletePdb(tfjob); err != nil {
				tc.Recorder.Eventf(tfjob, v1.EventTypeWarning, "FailedDeletePdb", "Error deleting: %v", err)
				return err
			} else {
				tc.Recorder.Eventf(tfjob, v1.EventTypeNormal, "SuccessfulDeletePdb", "Deleted pdb: %v", tfjob.Name)

			}
		}

		// At this point the pods may have been deleted, so if the job succeeded, we need to manually set the replica status.
		// If any replicas are still Active, set their status to succeeded.
		if isSucceeded(tfjob.Status) {
			for rtype := range tfjob.Status.ReplicaStatuses {
				tfjob.Status.ReplicaStatuses[rtype].Succeeded += tfjob.Status.ReplicaStatuses[rtype].Active
				tfjob.Status.ReplicaStatuses[rtype].Active = 0
			}
		}
		return tc.updateStatusHandler(tfjob)
	}

	// Save the current state of the replicas
	replicasStatus := make(map[string]v1.PodPhase)

	// Diff current active pods/services with replicas.
	for rtype, spec := range tfjob.Spec.TFReplicaSpecs {
		err = tc.reconcilePods(tfjob, pods, rtype, spec, replicasStatus)
		if err != nil {
			logger.Warnf("reconcilePods error %v", err)
			return err
		}

		err = tc.reconcileServices(tfjob, services, rtype, spec)

		if err != nil {
			logger.Warnf("reconcileServices error %v", err)
			return err
		}
	}

	// TODO(CPH): Add check here, no need to update the tfjob if the status hasn't changed since last time.
	return tc.updateStatusHandler(tfjob)
}

// satisfiedExpectations returns true if the required adds/dels for the given tfjob have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (tc *TFController) satisfiedExpectations(tfjob *tfv1beta1.TFJob) bool {
	satisfied := false
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfjob, err))
		return false
	}

	for rtype := range tfjob.Spec.TFReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := jobcontroller.GenExpectationPodsKey(tfjobKey, string(rtype))
		satisfied = satisfied || tc.Expectations.SatisfiedExpectations(expectationPodsKey)

		// Check the expectations of the services.
		expectationServicesKey := jobcontroller.GenExpectationServicesKey(tfjobKey, string(rtype))
		satisfied = satisfied || tc.Expectations.SatisfiedExpectations(expectationServicesKey)
	}

	return satisfied
}

func (tc *TFController) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	return tc.getTFJobFromName(namespace, name)
}

func (tc *TFController) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	return tc.tfJobClientSet.KubeflowV1beta1().TFJobs(namespace).Get(name, metav1.GetOptions{})
}

func (tc *TFController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return tfv1beta1.SchemeGroupVersionKind
}

func (tc *TFController) GetAPIGroupVersion() schema.GroupVersion {
	return tfv1beta1.SchemeGroupVersion
}

func (tc *TFController) GetGroupNameLabelKey() string {
	return labelGroupName
}

func (tc *TFController) GetJobNameLabelKey() string {
	return labelTFJobName
}

func (tc *TFController) GetGroupNameLabelValue() string {
	return tfv1beta1.GroupName
}

func (tc *TFController) GetReplicaTypeLabelKey() string {
	return tfReplicaTypeLabel
}

func (tc *TFController) GetReplicaIndexLabelKey() string {
	return tfReplicaIndexLabel
}

func (tc *TFController) ControllerName() string {
	return controllerName
}
