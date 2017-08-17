// package controller provides a Kubernetes controller for a TensorFlow job resource.
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"k8s.io/client-go/kubernetes"
	"github.com/jlewi/mlkube.io/pkg/spec"
	"github.com/jlewi/mlkube.io/pkg/trainer"
	"github.com/jlewi/mlkube.io/pkg/util/k8sutil"
	"net/http"
	"reflect"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	log "github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwatch "k8s.io/apimachinery/pkg/watch"
	k8sErrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	ErrVersionOutdated = errors.New("requested version is outdated in apiserver")

	initRetryWaitTime = 30 * time.Second

	// Workaround for watching TPR resource.
	// client-go has encoding issue and we want something more predictable.
	KubeHttpCli *http.Client
	MasterHost  string
)

type Event struct {
	Type   kwatch.EventType
	Object *spec.TfJob
}

type Controller struct {
	Namespace   string
	KubeCli     kubernetes.Interface
	ApiCli      apiextensionsclient.Interface
	TfJobClient k8sutil.TfJobClient

	config spec.ControllerConfig
	jobs   map[string]*trainer.TrainingJob
	// Kubernetes resource version of the jobs
	jobRVs    map[string]string
	stopChMap map[string]chan struct{}

	// TODO(jlewi): waitJobs should probably be used to ensure TrainingJob has finished processing
	// a stop event before shutting down and deleting all jobs.
	waitJobs sync.WaitGroup
}

func New(kubeCli kubernetes.Interface, apiCli apiextensionsclient.Interface, tfJobClient k8sutil.TfJobClient, ns string, config spec.ControllerConfig) *Controller {
	if tfJobClient == nil {
		panic("tfJobClient can't be nil")
	}
	return &Controller{
		Namespace:   ns,
		KubeCli:     kubeCli,
		ApiCli:  		apiCli,
		TfJobClient: tfJobClient,
		// TODO(jlewi)): What to do about cluster.Cluster?
		jobs:      make(map[string]*trainer.TrainingJob),
		jobRVs:    make(map[string]string),
		stopChMap: map[string]chan struct{}{},
		config:    config,
	}
}

func (c *Controller) Run() error {
	var (
		watchVersion string
		err          error
	)

	for {
		watchVersion, err = c.initResource()
		if err == nil {
			break
		}
		log.Errorf("initialization failed: %v", err)
		log.Infof("retry in %v...", initRetryWaitTime)
		time.Sleep(initRetryWaitTime)
		// todo: add max retry?
	}

	log.Infof("starts running from watch version: %s", watchVersion)

	defer func() {
		for _, stopC := range c.stopChMap {
			close(stopC)
		}
		c.waitJobs.Wait()
	}()

	eventCh, errCh := c.watch(watchVersion)

	go func() {
		pt := newPanicTimer(time.Minute, "unexpected long blocking (> 1 Minute) when handling cluster event")

		for ev := range eventCh {
			pt.start()
			if err := c.handleClusterEvent(ev); err != nil {
				log.Warningf("fail to handle event: %v", err)
			}
			pt.stop()
		}
	}()
	return <-errCh
}

func (c *Controller) handleClusterEvent(event *Event) error {
	clus := event.Object

	if clus.Status.IsFailed() {
		if event.Type == kwatch.Deleted {
			delete(c.jobs, clus.Metadata.Name)
			delete(c.jobRVs, clus.Metadata.Name)
			return nil
		}
		return fmt.Errorf("ignore failed cluster (%s). Please delete its TPR", clus.Metadata.Name)
	}

	// TODO: add validation to spec update.
	clus.Spec.Cleanup()
	//
	switch event.Type {
	case kwatch.Added:
		// Event indicates that a new instance of the Cluster TPR was created.
		// So we create a Cluster object to control this resource.
		stopC := make(chan struct{})
		nc, err := trainer.NewJob(c.KubeCli, c.TfJobClient, clus, stopC, &c.waitJobs, &c.config)

		if err != nil {
			return err
		}
		//NewJob(kubeCli kubernetes.Interface, job spec.TfJob, stopC <-chan struct{}, wg *sync.WaitGroup)

		c.stopChMap[clus.Metadata.Name] = stopC
		c.jobs[clus.Metadata.Name] = nc
		c.jobRVs[clus.Metadata.Name] = clus.Metadata.ResourceVersion

	//case kwatch.Modified:
	//  if _, ok := c.jobs[clus.Metadata.Name]; !ok {
	//    return fmt.Errorf("unsafe state. cluster was never created but we received event (%s)", event.Type)
	//  }
	//  c.jobs[clus.Metadata.Name].Update(clus)
	//  c.jobRVs[clus.Metadata.Name] = clus.Metadata.ResourceVersion
	//
	case kwatch.Deleted:
		if _, ok := c.jobs[clus.Metadata.Name]; !ok {
			return fmt.Errorf("unsafe state. TfJob was never created but we received event (%s)", event.Type)
		}
		c.jobs[clus.Metadata.Name].Delete()
		delete(c.jobs, clus.Metadata.Name)
		delete(c.jobRVs, clus.Metadata.Name)
	}
	return nil
}

func (c *Controller) findAllTfJobs() (string, error) {
	// TODO(jlewi): Need to implement this function.
	log.Info("finding existing jobs...")
	jobList, err := c.TfJobClient.List(c.Namespace)
	if err != nil {
		return "", err
	}

	for i := range jobList.Items {
		clus := jobList.Items[i]

		if clus.Status.IsFailed() {
			log.Infof("ignore failed TfJob (%s). Please delete its TPR", clus.Metadata.Name)
			continue
		}

		clus.Spec.Cleanup()

		stopC := make(chan struct{})
		nc, err := trainer.NewJob(c.KubeCli, c.TfJobClient, &clus, stopC, &c.waitJobs, &c.config)

		if err != nil {
			log.Errorf("traininer.NewJob() returned error; %v for job: %v", err, clus.Metadata.Name)
			continue
		}
		c.stopChMap[clus.Metadata.Name] = stopC
		c.jobs[clus.Metadata.Name] = nc
		c.jobRVs[clus.Metadata.Name] = clus.Metadata.ResourceVersion
	}

	return jobList.Metadata.ResourceVersion, nil
}

// makeClusterConfig creates the Config object from a cluster initializing it with data from the
// controller.
//func (c *Controller) makeClusterConfig() cluster.Config {
//  return cluster.Config{
//    // TODO(jlewi): Do we need a service account?
//    //ServiceAccount: c.Config.ServiceAccount,
//    KubeCli: c.KubeCli,
//  }
//}

func (c *Controller) initResource() (string, error) {
	watchVersion := "0"
	err := c.createTPR()
	if err != nil {
		if k8sutil.IsKubernetesResourceAlreadyExistError(err) {
			// TPR has been initialized before. We need to recover existing cluster.
			watchVersion, err = c.findAllTfJobs()
			if err != nil {
				log.Errorf("initResource() failed; findAllTfJobs returned error: %v", err)
				return "", err
			}
		} else {
			log.Errorf("createTPR() returned error: %v", err)
			return "", fmt.Errorf("fail to create TPR: %v", err)
		}
	}
	return watchVersion, nil
}

func (c *Controller) createTPR() error {
	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.TPRName(),
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: spec.TPRGroup,
			Version: spec.TPRVersion,
			 Scope: v1beta1.NamespaceScoped,
				Names: v1beta1.CustomResourceDefinitionNames{
					Plural: spec.TPRKindPlural,
					// TODO(jlewi): Do we want to set the singular name?
					// Kind is the serialized kind of the resource.  It is normally CamelCase and singular.
					Kind:   reflect.TypeOf(spec.TfJob{}).Name(),
				},
			},
	}

	_, err := c.ApiCli.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = c.ApiCli.ApiextensionsV1beta1().CustomResourceDefinitions().Get(spec.TPRName(), metav1.GetOptions{})
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
					log.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})

	if err != nil {
		deleteErr := c.ApiCli.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(spec.TPRName(), nil)
		if deleteErr != nil {
			return k8sErrors.NewAggregate([]error{err, deleteErr})
		}
		return err
	}
	return nil
}

// watch creates a go routine, and watches the TF cluster kind resources from
// the given watch version. It emits events on the resources through the returned
// event chan. Errors will be reported through the returned error chan. The go routine
// exits on any error.
func (c *Controller) watch(watchVersion string) (<-chan *Event, <-chan error) {
	eventCh := make(chan *Event)
	// On unexpected error case, controller should exit
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		for {
			resp, err := c.TfJobClient.Watch(MasterHost, c.Namespace, KubeHttpCli, watchVersion)
			if err != nil {
				errCh <- err
				return
			}
			if resp.StatusCode != http.StatusOK {
				log.Infof("WatchClusters response: %+v", resp)
				resp.Body.Close()
				errCh <- errors.New("invalid status code: " + resp.Status)
				return
			}

			log.Infof("start watching at %v", watchVersion)

			decoder := json.NewDecoder(resp.Body)
			for {
				ev, st, err := pollEvent(decoder)
				if err != nil {
					if err == io.EOF { // apiserver will close stream periodically
						log.Info("apiserver closed stream")
						break
					}

					log.Errorf("received invalid event from API server: %v", err)
					errCh <- err
					return
				}

				if st != nil {
					resp.Body.Close()

					if st.Code == http.StatusGone {
						// event history is outdated.
						// if nothing has changed, we can go back to watch again.
						clusterList, err := c.TfJobClient.List(c.Namespace)
						if err == nil && !c.isClustersCacheStale(clusterList.Items) {
							watchVersion = clusterList.Metadata.ResourceVersion
							break
						}

						// if anything has changed (or error on relist), we have to rebuild the state.
						// go to recovery path
						errCh <- ErrVersionOutdated
						return
					}

					log.Fatalf("unexpected status response from API server: %v", st.Message)
				}

				log.Infof("event: %v %v", ev.Type, ev.Object.Spec)
				log.Infof("TfJob event: %v %v", ev.Type, ev.Object.Spec)

				watchVersion = ev.Object.Metadata.ResourceVersion
				eventCh <- ev
			}

			resp.Body.Close()
		}
	}()

	return eventCh, errCh
}

func (c *Controller) isClustersCacheStale(currentClusters []spec.TfJob) bool {
	if len(c.jobRVs) != len(currentClusters) {
		return true
	}

	for _, cc := range currentClusters {
		rv, ok := c.jobRVs[cc.Metadata.Name]
		if !ok || rv != cc.Metadata.ResourceVersion {
			return true
		}
	}

	return false
}
