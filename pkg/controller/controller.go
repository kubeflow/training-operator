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

// Package controller provides a Kubernetes controller for a TensorFlow job resource.
package controller

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/juju/ratelimit"
	tfv1alpha1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha1"
	tfjobclient "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	kubeflowscheme "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	informers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	listers "github.com/kubeflow/tf-operator/pkg/client/listers/kubeflow/v1alpha1"
	"github.com/kubeflow/tf-operator/pkg/trainer"
)

const (
	controllerName = "kubeflow"
)

var (
	// ErrVersionOutdated is a exported var to capture the error in apiserver
	ErrVersionOutdated = errors.New("requested version is outdated in apiserver")

	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// DefaultJobBackOff is the max backoff period, exported for the e2e test
	DefaultJobBackOff = 10 * time.Second
	// MaxJobBackOff is the max backoff period, exported for the e2e test
	MaxJobBackOff = 360 * time.Second
)

// Controller is structure to manage various service clients
type Controller struct {
	KubeClient  kubernetes.Interface
	TFJobClient tfjobclient.Interface

	config tfv1alpha1.ControllerConfig
	jobs   map[string]*trainer.TrainingJob

	TFJobLister listers.TFJobLister
	TFJobSynced cache.InformerSynced

	// WorkQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	//
	// Items in the work queue correspond to the name of the job.
	// In response to various events (e.g. Add, Update, Delete), the informer
	// is configured to add events to the queue. Since the item in the queue
	// represents a job and not a particular event, we end up aggregating events for
	// a job and ensure that a particular job isn't being processed by multiple
	// workers simultaneously.
	//
	// We rely on the informer to periodically generate Update events. This ensures
	// we regularly check on each TFJob and take any action needed.
	//
	// If there is a problem processing a job, processNextWorkItem just requeues
	// the work item. This ensures that we end up retrying it. In this case
	// we rely on the rateLimiter in the worker queue to retry with exponential
	// backoff.
	WorkQueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	syncHandler func(jobKey string) (bool, error)

	enableGangScheduling bool
}

// New method sets up service client handles and returns controller object
func New(kubeClient kubernetes.Interface, tfJobClient tfjobclient.Interface,
	config tfv1alpha1.ControllerConfig, tfJobInformerFactory informers.SharedInformerFactory,
	enableGangScheduling bool) (*Controller, error) {
	tfJobInformer := tfJobInformerFactory.Kubeflow().V1alpha1().TFJobs()

	kubeflowscheme.AddToScheme(scheme.Scheme)
	log.Debug("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName})

	// Use a ratelimiter with overall  and per-item rate limitting.
	// The overall is a token bucket and the per-item is exponential
	// For the per item
	rateLimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&workqueue.BucketRateLimiter{Bucket: ratelimit.NewBucketWithRate(float64(10), int64(100))},
	)

	controller := &Controller{
		KubeClient:  kubeClient,
		TFJobClient: tfJobClient,
		WorkQueue:   workqueue.NewNamedRateLimitingQueue(rateLimiter, "TFjobs"),
		recorder:    recorder,
		// TODO(jlewi)): What to do about cluster.Cluster?
		jobs:                 make(map[string]*trainer.TrainingJob),
		config:               config,
		enableGangScheduling: enableGangScheduling,
	}

	log.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	tfJobInformer.Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *tfv1alpha1.TFJob:
					log.Debugf("filter tfjob name: %v", t.Name)
					return true
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: controller.enqueueController,
				UpdateFunc: func(oldObj, newObj interface{}) {
					controller.enqueueController(newObj)
				},
				DeleteFunc: controller.enqueueController,
			},
		})

	controller.TFJobLister = tfJobInformer.Lister()
	controller.TFJobSynced = tfJobInformer.Informer().HasSynced
	controller.syncHandler = controller.syncTFJob

	return controller, nil
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.WorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting TFJob controller")

	// Wait for the caches to be synced before starting workers
	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.TFJobSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Infof("Starting %v workers", threadiness)
	// Launch workers to process TFJob resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	key, quit := c.WorkQueue.Get()
	if quit {
		return false
	}

	defer c.WorkQueue.Done(key)

	_, err := c.syncHandler(key.(string))
	if err == nil {
		// Calling forget resets the rate limiter for this item.
		// Since the sync was processed successfully we want to reset the ratelimiter
		// so that future events can be processed immediately.
		log.WithFields(log.Fields{
			"job": key,
		}).Infof("WorkQueue forgetting key %v", key)
		c.WorkQueue.Forget(key)
		return true
	}

	// There was an error processing the key so to retry we requeue it.
	// The WorkQueue uses a rate limiter to control when the key gets retried.
	utilruntime.HandleError(fmt.Errorf("Error syncing job: %v", err))
	c.WorkQueue.AddRateLimited(key)

	return true
}

// syncTFJob will sync the job with the given. This function is not meant to be invoked
// concurrently with the same key.
//
// When a job is completely processed it will return true indicating that its ok to forget about this job since
// no more processing will occur for it.
func (c *Controller) syncTFJob(key string) (bool, error) {
	startTime := time.Now()
	defer func() {
		log.WithFields(log.Fields{
			"job": key,
		}).Infof("Finished syncing job %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	if len(ns) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid job key %q: either namespace or name is missing", key)
	}

	tfJob, err := c.TFJobLister.TFJobs(ns).Get(name)

	if err != nil {
		if apierrors.IsNotFound(err) {
			log.WithFields(log.Fields{
				"job": key,
			}).Infof("Job has been deleted: %v", key)
			return true, nil
		}
		return false, err
	}

	// Create a new TrainingJob if there is no TrainingJob stored for it in the jobs map or if the UID's don't match.
	// The UID's won't match in the event we deleted the job and then recreated the job with the same name.
	if cJob, ok := c.jobs[key]; !ok || cJob.UID() != tfJob.UID {
		log.WithFields(log.Fields{
			"job": key,
		}).Infof("Creating new job %v", key)
		nc, err := trainer.NewJob(c.KubeClient, c.TFJobClient, c.recorder, tfJob, &c.config)

		if err != nil {
			log.WithFields(log.Fields{
				"job": key,
			}).Errorf("There was a problem creating NewJob %v; Error: %v", key, err)
			return false, err
		}
		c.jobs[key] = nc
	} else {
		// Replace the TFJob stored inside TrainingJob with the latest job.
		// We need to do this to pull in the latest changes to the spec/status.
		c.jobs[key].Update(tfJob)
	}

	nc := c.jobs[key]

	if err := nc.Reconcile(&c.config, c.enableGangScheduling); err != nil {
		return false, err
	}

	// TODO(jlewi): Why do we issue a get request again here?
	tfJob, err = c.TFJobClient.KubeflowV1alpha1().TFJobs(tfJob.ObjectMeta.Namespace).Get(tfJob.ObjectMeta.Name, metav1.GetOptions{})

	if err != nil {
		return false, err
	}

	// TODO(jlewi): This logic will need to change when/if we get rid of phases and move to conditions. At that
	// case we should forget about a job when the appropriate condition is reached.
	if tfJob.Status.Phase == tfv1alpha1.TFJobPhaseCleanUp {
		return true, nil
	}
	return false, nil

}

// obj could be an *batch.Job, or a DeletionFinalStateUnknown marker item.
func (c *Controller) enqueueController(obj interface{}) {
	key, err := keyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %+v: %v", obj, err))
		return
	}

	c.WorkQueue.AddRateLimited(key)
}
