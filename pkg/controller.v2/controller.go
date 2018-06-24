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
package controller

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
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobscheme "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	tfjobinformersv1alpha2 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/kubeflow/v1alpha2"
	tfjoblisters "github.com/kubeflow/tf-operator/pkg/client/listers/kubeflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/control"
)

const (
	controllerName = "tf-operator"

	// labels for pods and servers.
	tfReplicaTypeLabel  = "tf-replica-type"
	tfReplicaIndexLabel = "tf-replica-index"
)

var (
	// controllerKind is GroupVersionKind for this controller type.
	controllerKind = tfv1alpha2.SchemeGroupVersionKind

	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// DefaultTFJobControllerConfiguration is the suggested tf-operator configuration for production.
	DefaultTFJobControllerConfiguration = TFJobControllerConfiguration{
		ReconcilerSyncLoopPeriod: metav1.Duration{Duration: 15 * time.Second},
	}
)

// TFJobControllerConfiguration contains configuration of tf-operator.
// DefaultTimerConfig is the suggested tf-operator configuration for production.
type TFJobControllerConfiguration struct {
	// ReconcilerSyncLoopPeriod is the amount of time the reconciler sync states loop
	// wait between two reconciler sync.
	// It is set to 15 sec by default.
	// TODO(cph): maybe we can let it grows by multiple in the future
	// and up to 5 minutes to reduce idle loop.
	// e.g. 15s, 30s, 60s, 120s...
	ReconcilerSyncLoopPeriod metav1.Duration
}

// TFJobController is the type for TFJob Controller, which manages
// the lifecycle of TFJobs.
type TFJobController struct {
	config TFJobControllerConfiguration

	// podControl is used to add or delete pods.
	podControl controller.PodControlInterface

	// serviceControl is used to add or delete services.
	serviceControl control.ServiceControlInterface

	// kubeClientSet is a standard kubernetes clientset.
	kubeClientSet kubeclientset.Interface

	// tfJobClientSet is a clientset for CRD TFJob.
	tfJobClientSet tfjobclientset.Interface

	// To allow injection of syncTFJob for testing.
	syncHandler func(tfJobKey string) (bool, error)

	// To allow injection of updateStatus for testing.
	updateStatusHandler func(tfjob *tfv1alpha2.TFJob) error

	// tfJobInformer is a temporary field for unstructured informer support.
	tfJobInformer cache.SharedIndexInformer

	// Listers for TFJob, Pod and Service
	// tfJobLister can list/get tfjobs from the shared informer's store.
	tfJobLister tfjoblisters.TFJobLister

	// podLister can list/get pods from the shared informer's store.
	podLister corelisters.PodLister

	// serviceLister can list/get services from the shared informer's store.
	serviceLister corelisters.ServiceLister

	// tfJobInformerSynced returns true if the tfjob store has been synced at least once.
	tfJobInformerSynced cache.InformerSynced

	// podInformerSynced returns true if the pod store has been synced at least once.
	podInformerSynced cache.InformerSynced

	// serviceInformerSynced returns true if the service store has been synced at least once.
	serviceInformerSynced cache.InformerSynced

	// A TTLCache of pod/services creates/deletes each tfjob expects to see
	// We use TFJob namespace/name + TFReplicaType + pods/services as an expectation key,
	// For example, there is a TFJob with namespace "tf-operator" and name "tfjob-abc":
	// {
	//     "PS": {
	//         "Replicas": 2,
	//     },
	//     "Worker": {
	//         "Replicas": 4,
	//     }
	// }
	// We will create 4 expectations:
	// - "tf-operator/tfjob-abc/ps/services", expects 2 adds.
	// - "tf-operator/tfjob-abc/ps/pods", expects 2 adds.
	// - "tf-operator/tfjob-abc/worker/services", expects 4 adds.
	// - "tf-operator/tfjob-abc/worker/pods", expects 4 adds.
	expectations controller.ControllerExpectationsInterface

	// workQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workQueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewTFJobController returns a new TFJob controller.
func NewTFJobController(
	// This variable is for unstructured informer.
	tfJobInformer tfjobinformersv1alpha2.TFJobInformer,
	kubeClientSet kubeclientset.Interface,
	tfJobClientSet tfjobclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	// This field is not used now but we keep it since it will be used
	// after we support CRD validation.
	tfJobInformerFactory tfjobinformers.SharedInformerFactory) *TFJobController {

	tfjobscheme.AddToScheme(scheme.Scheme)

	log.Debug("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName})

	realPodControl := control.RealPodControl{
		KubeClient: kubeClientSet,
		Recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName}),
	}

	realServiceControl := control.RealServiceControl{
		KubeClient: kubeClientSet,
		Recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName}),
	}

	// Create new TFJobController.
	tc := &TFJobController{
		podControl:     realPodControl,
		serviceControl: realServiceControl,
		kubeClientSet:  kubeClientSet,
		tfJobClientSet: tfJobClientSet,
		expectations:   controller.NewControllerExpectations(),
		workQueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), tfv1alpha2.Plural),
		recorder:       recorder,
	}

	// Set sync handler.
	tc.syncHandler = tc.syncTFJob
	tc.updateStatusHandler = tc.updateTFJobStatus

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
		AddFunc:    tc.addPod,
		UpdateFunc: tc.updatePod,
		DeleteFunc: tc.deletePod,
	})

	tc.podLister = podInformer.Lister()
	tc.podInformerSynced = podInformer.Informer().HasSynced

	// Create service informer.
	serviceInformer := kubeInformerFactory.Core().V1().Services()

	// Set up an event handler for when service resources change.
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    tc.addService,
		UpdateFunc: tc.updateService,
		DeleteFunc: tc.deleteService,
	})

	tc.serviceLister = serviceInformer.Lister()
	tc.serviceInformerSynced = serviceInformer.Informer().HasSynced

	return tc
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (tc *TFJobController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer tc.workQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches.
	log.Info("Starting TFJob controller")

	// Wait for the caches to be synced before starting workers.
	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, tc.tfJobInformerSynced); !ok {
		return fmt.Errorf("failed to wait for tfjob caches to sync")
	}

	if ok := cache.WaitForCacheSync(stopCh, tc.podInformerSynced); !ok {
		return fmt.Errorf("failed to wait for pod caches to sync")
	}

	if ok := cache.WaitForCacheSync(stopCh, tc.serviceInformerSynced); !ok {
		return fmt.Errorf("failed to wait for service caches to sync")
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
func (tc *TFJobController) runWorker() {
	for tc.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (tc *TFJobController) processNextWorkItem() bool {
	key, quit := tc.workQueue.Get()
	if quit {
		return false
	}
	defer tc.workQueue.Done(key)

	tfJob, err := tc.getTFJobFromKey(key.(string))
	if err != nil {
		log.Errorf("Failed to get TFJob from key %s: %v", key, err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to TFJob object: %v", err)
			loggerForTFJob(tfJob).Warn(errMsg)
			tc.recorder.Event(tfJob, v1.EventTypeWarning, failedMarshalTFJobReason, errMsg)
		}
		return true
	}

	// Sync TFJob to match the actual state to this desired state.
	forget, err := tc.syncHandler(key.(string))
	if err == nil {
		if forget {
			tc.workQueue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing tfjob: %v", err))
	tc.workQueue.AddRateLimited(key)

	return true
}

func (tc *TFJobController) enqueueTFJob(tfjob interface{}) {
	key, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfjob, err))
		return
	}

	// TODO: we may need add backoff here
	tc.workQueue.Add(key)
}

// syncTFJob syncs the tfjob with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods/services created or deleted.
// This function is not meant to be invoked concurrently with the same key.
func (tc *TFJobController) syncTFJob(key string) (bool, error) {
	startTime := time.Now()
	defer func() {
		log.Infof("Finished syncing tfjob %q (%v)", key, time.Since(startTime))
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
			log.Infof("TFJob has been deleted: %v", key)
			// jm.expectations.DeleteExpectations(key)
			return true, nil
		}
		return false, err
	}

	tfjob := sharedTFJob.DeepCopy()
	tfjobNeedsSync := tc.satisfiedExpectations(tfjob)

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

// reconcileTFJobs checks and updates replicas for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting pods/services.
func (tc *TFJobController) reconcileTFJobs(tfjob *tfv1alpha2.TFJob) error {
	log.Infof("Reconcile TFJobs %s", tfjob.Name)

	pods, err := tc.getPodsForTFJob(tfjob)

	if err != nil {
		log.Infof("getPodsForTFJob error %v", err)
		return err
	}

	services, err := tc.getServicesForTFJob(tfjob)

	if err != nil {
		log.Infof("getServicesForTFJob error %v", err)
		return err
	}

	// If the TFJob is terminated, delete all pods and services.
	if isSucceeded(tfjob.Status) || isFailed(tfjob.Status) {
		if err := tc.deletePodsAndServices(tfjob, pods); err != nil {
			return err
		}
		// Initialize the status.
		initializeTFReplicaStatuses(tfjob, tfv1alpha2.TFReplicaTypeWorker)
		initializeTFReplicaStatuses(tfjob, tfv1alpha2.TFReplicaTypePS)
		initializeTFReplicaStatuses(tfjob, tfv1alpha2.TFReplicaTypeChief)
		return tc.updateStatusHandler(tfjob)
	}

	// Save the current state of the replicas
	replicasStatus := make(map[string]v1.PodPhase)

	// Diff current active pods/services with replicas.
	for rtype, spec := range tfjob.Spec.TFReplicaSpecs {
		err = tc.reconcilePods(tfjob, pods, rtype, spec, replicasStatus)
		if err != nil {
			log.Infof("reconcilePods error %v", err)
			return err
		}

		err = tc.reconcileServices(tfjob, services, rtype, spec)

		if err != nil {
			log.Infof("reconcileServices error %v", err)
			return err
		}
	}

	// TODO(CPH): Add check here, no need to update the tfjob if the status hasn't changed since last time.
	return tc.updateStatusHandler(tfjob)
}

// satisfiedExpectations returns true if the required adds/dels for the given tfjob have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (tc *TFJobController) satisfiedExpectations(tfjob *tfv1alpha2.TFJob) bool {
	satisfied := false
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfjob, err))
		return false
	}

	for rtype := range tfjob.Spec.TFReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := genExpectationPodsKey(tfjobKey, string(rtype))
		satisfied = satisfied || tc.expectations.SatisfiedExpectations(expectationPodsKey)

		// Check the expectations of the services.
		expectationServicesKey := genExpectationServicesKey(tfjobKey, string(rtype))
		satisfied = satisfied || tc.expectations.SatisfiedExpectations(expectationServicesKey)
	}

	return satisfied
}

// resolveControllerRef returns the tfjob referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching tfjob
// of the correct Kind.
func (tc *TFJobController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) *tfv1alpha2.TFJob {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != controllerKind.Kind {
		return nil
	}
	tfjob, err := tc.getTFJobFromName(namespace, controllerRef.Name)
	if err != nil {
		return nil
	}
	if tfjob.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return tfjob
}
