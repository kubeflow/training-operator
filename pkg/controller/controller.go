// package controller provides a Kubernetes controller for a TensorFlow job resource.
package controller

import (
	"errors"
	"fmt"

	"reflect"
	"time"

	"github.com/tensorflow/k8s/pkg/trainer"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/golang/glog"
	"github.com/tensorflow/k8s/pkg/apis/tensorflow/helper"
	tfv1alpha1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	tfjobclient "github.com/tensorflow/k8s/pkg/client/clientset/versioned"
	informers "github.com/tensorflow/k8s/pkg/client/informers/externalversions"
	listers "github.com/tensorflow/k8s/pkg/client/listers/tensorflow/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sErrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	ErrVersionOutdated = errors.New("requested version is outdated in apiserver")

	keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// DefaultJobBackOff is the max backoff period, exported for the e2e test
	DefaultJobBackOff = 10 * time.Second
	// MaxJobBackOff is the max backoff period, exported for the e2e test
	MaxJobBackOff = 360 * time.Second
)

type Controller struct {
	KubeClient   kubernetes.Interface
	ApiExtclient apiextensionsclient.Interface
	TfJobClient  tfjobclient.Interface

	config tfv1alpha1.ControllerConfig
	jobs   map[string]*trainer.TrainingJob

	TfJobLister listers.TfJobLister
	TfJobSynced cache.InformerSynced

	// WorkQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	WorkQueue workqueue.RateLimitingInterface

	syncHandler func(jobKey string) (bool, error)
}

func New(kubeClient kubernetes.Interface, apiExtclient apiextensionsclient.Interface, tfJobClient tfjobclient.Interface,
	config tfv1alpha1.ControllerConfig, tfJobInformerFactory informers.SharedInformerFactory) (*Controller, error) {
	tfJobInformer := tfJobInformerFactory.Tensorflow().V1alpha1().TfJobs()

	controller := &Controller{
		KubeClient:   kubeClient,
		ApiExtclient: apiExtclient,
		TfJobClient:  tfJobClient,
		WorkQueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Tfjobs"),
		// TODO(jlewi)): What to do about cluster.Cluster?
		jobs:   make(map[string]*trainer.TrainingJob),
		config: config,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	tfJobInformer.Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *tfv1alpha1.TfJob:
					glog.V(4).Infof("filter tfjob name: %v", t.Name)
					return true
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    controller.enqueueController,
				UpdateFunc: func(oldObj, newObj interface{}) {
					controller.enqueueController(newObj)
				},
				DeleteFunc: controller.handleDelete,
			},
		})

	controller.TfJobLister = tfJobInformer.Lister()
	controller.TfJobSynced = tfJobInformer.Informer().HasSynced
	controller.syncHandler = controller.syncTFJob

	if err := controller.createCRD(); err != nil {
		return nil, err
	}

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
	glog.Info("Starting TfJob controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.TfJobSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

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

	forget, err := c.syncHandler(key.(string))
	if err == nil {
		if forget {
			c.WorkQueue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing job: %v", err))
	c.WorkQueue.AddRateLimited(key)

	return true
}

// syncJob will sync the job with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods created or deleted. This function is not meant to be invoked
// concurrently with the same key.
func (c *Controller) syncTFJob(key string) (bool, error) {
	startTime := time.Now()
	defer func() {
		glog.V(4).Infof("Finished syncing job %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	if len(ns) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid job key %q: either namespace or name is missing", key)
	}

	tfJob, err := c.TfJobLister.TfJobs(ns).Get(name)

	if err != nil {
		if apierrors.IsNotFound(err) {
			glog.V(4).Infof("Job has been deleted: %v", key)
			return true, nil
		}
		return false, err
	}

	if _, ok := c.jobs[tfJob.ObjectMeta.Namespace+"-"+tfJob.ObjectMeta.Name]; !ok {
		nc, err := trainer.NewJob(c.KubeClient, c.TfJobClient, tfJob, &c.config)

		if err != nil {
			return false, err
		}
		c.jobs[tfJob.ObjectMeta.Namespace+"-"+tfJob.ObjectMeta.Name] = nc
	}

	nc := c.jobs[tfJob.ObjectMeta.Namespace+"-"+tfJob.ObjectMeta.Name]

	if err := nc.Reconcile(&c.config); err != nil {
		return false, err
	}

	tfJob, err = c.TfJobClient.TensorflowV1alpha1().TfJobs(tfJob.ObjectMeta.Namespace).Get(tfJob.ObjectMeta.Name, metav1.GetOptions{})

	if err != nil {
		return false, err
	}

	if tfJob.Status.State == tfv1alpha1.StateSucceeded {
		return true, nil
	} else {
		return false, nil
	}

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

func (c *Controller) handleDelete(obj interface{}) {
	tfjob := obj.(*tfv1alpha1.TfJob)
	if _, ok := c.jobs[tfjob.ObjectMeta.Namespace+"-"+tfjob.ObjectMeta.Name]; !ok {
		glog.V(4).Infof("unsafe state. TfJob was never created but we received delete event")
		return
	}

	c.jobs[tfjob.ObjectMeta.Namespace+"-"+tfjob.ObjectMeta.Name].Delete()
}

func (c *Controller) createCRD() error {
	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: helper.CRDName(),
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   tfv1alpha1.CRDGroup,
			Version: tfv1alpha1.CRDVersion,
			Scope:   v1beta1.NamespaceScoped,
			Names: v1beta1.CustomResourceDefinitionNames{
				Plural:   tfv1alpha1.CRDKindPlural,
				Singular: tfv1alpha1.CRDKind,
				// Kind is the serialized kind of the resource.  It is normally CamelCase and singular.
				Kind: reflect.TypeOf(tfv1alpha1.TfJob{}).Name(),
			},
		},
	}

	_, err := c.ApiExtclient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = c.ApiExtclient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(helper.CRDName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case v1beta1.Established:
				if cond.Status == v1beta1.ConditionTrue {
					return true, err
				}
			case v1beta1.NamesAccepted:
				if cond.Status == v1beta1.ConditionFalse {
					glog.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})

	if err != nil {
		deleteErr := c.ApiExtclient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(helper.CRDName(), nil)
		if deleteErr != nil {
			return k8sErrors.NewAggregate([]error{err, deleteErr})
		}
		return err
	}
	return nil
}
