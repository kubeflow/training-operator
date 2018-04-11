/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note(CPH): this file is copied form k8s.io/kubernetes/pkg/controller
// We should not import the huge package k8s.io/kubernetes/pkg
package controller

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/golang/glog"
)

const (
	// If a watch drops a delete event for a pod, it'll take this long
	// before a dormant controller waiting for those packets is woken up anyway. It is
	// specifically targeted at the case where some problem prevents an update
	// of expectations, without it the controller could stay asleep forever. This should
	// be set based on the expected latency of watch events.
	//
	// Currently a controller can service (create *and* observe the watch events for said
	// creation) about 10 pods a second, so it takes about 1 min to service
	// 500 pods. Just creation is limited to 20qps, and watching happens with ~10-30s
	// latency/pod at the scale of 3000 pods over 100 nodes.
	ExpectationsTimeout = 5 * time.Minute
	// When batching pod creates, SlowStartInitialBatchSize is the size of the
	// initial batch.  The size of each successive batch is twice the size of
	// the previous batch.  For example, for a value of 1, batch sizes would be
	// 1, 2, 4, 8, ...  and for a value of 10, batch sizes would be
	// 10, 20, 40, 80, ...  Setting the value higher means that quota denials
	// will result in more doomed API calls and associated event spam.  Setting
	// the value lower will result in more API call round trip periods for
	// large batches.
	//
	// Given a number of pods to start "N":
	// The number of doomed calls per sync once quota is exceeded is given by:
	//      min(N,SlowStartInitialBatchSize)
	// The number of batches is given by:
	//      1+floor(log_2(ceil(N/SlowStartInitialBatchSize)))
	SlowStartInitialBatchSize = 1
)

var UpdateTaintBackoff = wait.Backoff{
	Steps:    5,
	Duration: 100 * time.Millisecond,
	Jitter:   1.0,
}

var (
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

type ResyncPeriodFunc func() time.Duration

// Returns 0 for resyncPeriod in case resyncing is not needed.
func NoResyncPeriodFunc() time.Duration {
	return 0
}

// StaticResyncPeriodFunc returns the resync period specified
func StaticResyncPeriodFunc(resyncPeriod time.Duration) ResyncPeriodFunc {
	return func() time.Duration {
		return resyncPeriod
	}
}

// Expectations are a way for controllers to tell the controller manager what they expect. eg:
//	ControllerExpectations: {
//		controller1: expects  2 adds in 2 minutes
//		controller2: expects  2 dels in 2 minutes
//		controller3: expects -1 adds in 2 minutes => controller3's expectations have already been met
//	}
//
// Implementation:
//	ControlleeExpectation = pair of atomic counters to track controllee's creation/deletion
//	ControllerExpectationsStore = TTLStore + a ControlleeExpectation per controller
//
// * Once set expectations can only be lowered
// * A controller isn't synced till its expectations are either fulfilled, or expire
// * Controllers that don't set expectations will get woken up for every matching controllee

// ExpKeyFunc to parse out the key from a ControlleeExpectation
var ExpKeyFunc = func(obj interface{}) (string, error) {
	if e, ok := obj.(*ControlleeExpectations); ok {
		return e.key, nil
	}
	return "", fmt.Errorf("Could not find key for obj %#v", obj)
}

// ControllerExpectationsInterface is an interface that allows users to set and wait on expectations.
// Only abstracted out for testing.
// Warning: if using KeyFunc it is not safe to use a single ControllerExpectationsInterface with different
// types of controllers, because the keys might conflict across types.
type ControllerExpectationsInterface interface {
	GetExpectations(controllerKey string) (*ControlleeExpectations, bool, error)
	SatisfiedExpectations(controllerKey string) bool
	DeleteExpectations(controllerKey string)
	SetExpectations(controllerKey string, add, del int) error
	ExpectCreations(controllerKey string, adds int) error
	ExpectDeletions(controllerKey string, dels int) error
	CreationObserved(controllerKey string)
	DeletionObserved(controllerKey string)
	RaiseExpectations(controllerKey string, add, del int)
	LowerExpectations(controllerKey string, add, del int)
}

// ControllerExpectations is a cache mapping controllers to what they expect to see before being woken up for a sync.
type ControllerExpectations struct {
	cache.Store
}

// GetExpectations returns the ControlleeExpectations of the given controller.
func (r *ControllerExpectations) GetExpectations(controllerKey string) (*ControlleeExpectations, bool, error) {
	if exp, exists, err := r.GetByKey(controllerKey); err == nil && exists {
		return exp.(*ControlleeExpectations), true, nil
	} else {
		return nil, false, err
	}
}

// DeleteExpectations deletes the expectations of the given controller from the TTLStore.
func (r *ControllerExpectations) DeleteExpectations(controllerKey string) {
	if exp, exists, err := r.GetByKey(controllerKey); err == nil && exists {
		if err := r.Delete(exp); err != nil {
			glog.V(2).Infof("Error deleting expectations for controller %v: %v", controllerKey, err)
		}
	}
}

// SatisfiedExpectations returns true if the required adds/dels for the given controller have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (r *ControllerExpectations) SatisfiedExpectations(controllerKey string) bool {
	if exp, exists, err := r.GetExpectations(controllerKey); exists {
		if exp.Fulfilled() {
			glog.V(4).Infof("Controller expectations fulfilled %#v", exp)
			return true
		} else if exp.isExpired() {
			glog.V(4).Infof("Controller expectations expired %#v", exp)
			return true
		} else {
			glog.V(4).Infof("Controller still waiting on expectations %#v", exp)
			return false
		}
	} else if err != nil {
		glog.V(2).Infof("Error encountered while checking expectations %#v, forcing sync", err)
	} else {
		// When a new controller is created, it doesn't have expectations.
		// When it doesn't see expected watch events for > TTL, the expectations expire.
		//	- In this case it wakes up, creates/deletes controllees, and sets expectations again.
		// When it has satisfied expectations and no controllees need to be created/destroyed > TTL, the expectations expire.
		//	- In this case it continues without setting expectations till it needs to create/delete controllees.
		glog.V(4).Infof("Controller %v either never recorded expectations, or the ttl expired.", controllerKey)
	}
	// Trigger a sync if we either encountered and error (which shouldn't happen since we're
	// getting from local store) or this controller hasn't established expectations.
	return true
}

// TODO: Extend ExpirationCache to support explicit expiration.
// TODO: Make this possible to disable in tests.
// TODO: Support injection of clock.
func (exp *ControlleeExpectations) isExpired() bool {
	return clock.RealClock{}.Since(exp.timestamp) > ExpectationsTimeout
}

// SetExpectations registers new expectations for the given controller. Forgets existing expectations.
func (r *ControllerExpectations) SetExpectations(controllerKey string, add, del int) error {
	exp := &ControlleeExpectations{add: int64(add), del: int64(del), key: controllerKey, timestamp: clock.RealClock{}.Now()}
	glog.V(4).Infof("Setting expectations %#v", exp)
	return r.Add(exp)
}

func (r *ControllerExpectations) ExpectCreations(controllerKey string, adds int) error {
	return r.SetExpectations(controllerKey, adds, 0)
}

func (r *ControllerExpectations) ExpectDeletions(controllerKey string, dels int) error {
	return r.SetExpectations(controllerKey, 0, dels)
}

// Decrements the expectation counts of the given controller.
func (r *ControllerExpectations) LowerExpectations(controllerKey string, add, del int) {
	if exp, exists, err := r.GetExpectations(controllerKey); err == nil && exists {
		exp.Add(int64(-add), int64(-del))
		// The expectations might've been modified since the update on the previous line.
		glog.V(4).Infof("Lowered expectations %#v", exp)
	}
}

// Increments the expectation counts of the given controller.
func (r *ControllerExpectations) RaiseExpectations(controllerKey string, add, del int) {
	if exp, exists, err := r.GetExpectations(controllerKey); err == nil && exists {
		exp.Add(int64(add), int64(del))
		// The expectations might've been modified since the update on the previous line.
		glog.V(4).Infof("Raised expectations %#v", exp)
	}
}

// CreationObserved atomically decrements the `add` expectation count of the given controller.
func (r *ControllerExpectations) CreationObserved(controllerKey string) {
	r.LowerExpectations(controllerKey, 1, 0)
}

// DeletionObserved atomically decrements the `del` expectation count of the given controller.
func (r *ControllerExpectations) DeletionObserved(controllerKey string) {
	r.LowerExpectations(controllerKey, 0, 1)
}

// Expectations are either fulfilled, or expire naturally.
type Expectations interface {
	Fulfilled() bool
}

// ControlleeExpectations track controllee creates/deletes.
type ControlleeExpectations struct {
	// Important: Since these two int64 fields are using sync/atomic, they have to be at the top of the struct due to a bug on 32-bit platforms
	// See: https://golang.org/pkg/sync/atomic/ for more information
	add       int64
	del       int64
	key       string
	timestamp time.Time
}

// Add increments the add and del counters.
func (e *ControlleeExpectations) Add(add, del int64) {
	atomic.AddInt64(&e.add, add)
	atomic.AddInt64(&e.del, del)
}

// Fulfilled returns true if this expectation has been fulfilled.
func (e *ControlleeExpectations) Fulfilled() bool {
	// TODO: think about why this line being atomic doesn't matter
	return atomic.LoadInt64(&e.add) <= 0 && atomic.LoadInt64(&e.del) <= 0
}

// GetExpectations returns the add and del expectations of the controllee.
func (e *ControlleeExpectations) GetExpectations() (int64, int64) {
	return atomic.LoadInt64(&e.add), atomic.LoadInt64(&e.del)
}

// NewControllerExpectations returns a store for ControllerExpectations.
func NewControllerExpectations() *ControllerExpectations {
	return &ControllerExpectations{cache.NewStore(ExpKeyFunc)}
}

// UIDSetKeyFunc to parse out the key from a UIDSet.
var UIDSetKeyFunc = func(obj interface{}) (string, error) {
	if u, ok := obj.(*UIDSet); ok {
		return u.key, nil
	}
	return "", fmt.Errorf("Could not find key for obj %#v", obj)
}

// UIDSet holds a key and a set of UIDs. Used by the
// UIDTrackingControllerExpectations to remember which UID it has seen/still
// waiting for.
type UIDSet struct {
	sets.String
	key string
}

// UIDTrackingControllerExpectations tracks the UID of the pods it deletes.
// This cache is needed over plain old expectations to safely handle graceful
// deletion. The desired behavior is to treat an update that sets the
// DeletionTimestamp on an object as a delete. To do so consistently, one needs
// to remember the expected deletes so they aren't double counted.
// TODO: Track creates as well (#22599)
type UIDTrackingControllerExpectations struct {
	ControllerExpectationsInterface
	// TODO: There is a much nicer way to do this that involves a single store,
	// a lock per entry, and a ControlleeExpectationsInterface type.
	uidStoreLock sync.Mutex
	// Store used for the UIDs associated with any expectation tracked via the
	// ControllerExpectationsInterface.
	uidStore cache.Store
}

// GetUIDs is a convenience method to avoid exposing the set of expected uids.
// The returned set is not thread safe, all modifications must be made holding
// the uidStoreLock.
func (u *UIDTrackingControllerExpectations) GetUIDs(controllerKey string) sets.String {
	if uid, exists, err := u.uidStore.GetByKey(controllerKey); err == nil && exists {
		return uid.(*UIDSet).String
	}
	return nil
}

// ExpectDeletions records expectations for the given deleteKeys, against the given controller.
func (u *UIDTrackingControllerExpectations) ExpectDeletions(rcKey string, deletedKeys []string) error {
	u.uidStoreLock.Lock()
	defer u.uidStoreLock.Unlock()

	if existing := u.GetUIDs(rcKey); existing != nil && existing.Len() != 0 {
		glog.Errorf("Clobbering existing delete keys: %+v", existing)
	}
	expectedUIDs := sets.NewString()
	for _, k := range deletedKeys {
		expectedUIDs.Insert(k)
	}
	glog.V(4).Infof("Controller %v waiting on deletions for: %+v", rcKey, deletedKeys)
	if err := u.uidStore.Add(&UIDSet{expectedUIDs, rcKey}); err != nil {
		return err
	}
	return u.ControllerExpectationsInterface.ExpectDeletions(rcKey, expectedUIDs.Len())
}

// DeletionObserved records the given deleteKey as a deletion, for the given rc.
func (u *UIDTrackingControllerExpectations) DeletionObserved(rcKey, deleteKey string) {
	u.uidStoreLock.Lock()
	defer u.uidStoreLock.Unlock()

	uids := u.GetUIDs(rcKey)
	if uids != nil && uids.Has(deleteKey) {
		glog.V(4).Infof("Controller %v received delete for pod %v", rcKey, deleteKey)
		u.ControllerExpectationsInterface.DeletionObserved(rcKey)
		uids.Delete(deleteKey)
	}
}

// DeleteExpectations deletes the UID set and invokes DeleteExpectations on the
// underlying ControllerExpectationsInterface.
func (u *UIDTrackingControllerExpectations) DeleteExpectations(rcKey string) {
	u.uidStoreLock.Lock()
	defer u.uidStoreLock.Unlock()

	u.ControllerExpectationsInterface.DeleteExpectations(rcKey)
	if uidExp, exists, err := u.uidStore.GetByKey(rcKey); err == nil && exists {
		if err := u.uidStore.Delete(uidExp); err != nil {
			glog.V(2).Infof("Error deleting uid expectations for controller %v: %v", rcKey, err)
		}
	}
}

// NewUIDTrackingControllerExpectations returns a wrapper around
// ControllerExpectations that is aware of deleteKeys.
func NewUIDTrackingControllerExpectations(ce ControllerExpectationsInterface) *UIDTrackingControllerExpectations {
	return &UIDTrackingControllerExpectations{ControllerExpectationsInterface: ce, uidStore: cache.NewStore(UIDSetKeyFunc)}
}

// Reasons for pod events
const (
	// FailedCreatePodReason is added in an event and in a replica set condition
	// when a pod for a replica set is failed to be created.
	FailedCreatePodReason = "FailedCreate"
	// SuccessfulCreatePodReason is added in an event when a pod for a replica set
	// is successfully created.
	SuccessfulCreatePodReason = "SuccessfulCreate"
	// FailedDeletePodReason is added in an event and in a replica set condition
	// when a pod for a replica set is failed to be deleted.
	FailedDeletePodReason = "FailedDelete"
	// SuccessfulDeletePodReason is added in an event when a pod for a replica set
	// is successfully deleted.
	SuccessfulDeletePodReason = "SuccessfulDelete"

	FailedCreateServiceReason     = "FailedCreateService"
	SuccessfulCreateServiceReason = "SuccessfulCreateService"
)

// PodControlInterface is an interface that knows how to add or delete pods
// created as an interface to allow testing.
type PodControlInterface interface {
	// CreatePods creates new pods according to the spec.
	CreatePods(namespace string, template *v1.PodTemplateSpec, object runtime.Object) error
	// CreatePodsOnNode creates a new pod according to the spec on the specified node,
	// and sets the ControllerRef.
	CreatePodsOnNode(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error
	// CreatePodsWithControllerRef creates new pods according to the spec, and sets object as the pod's controller.
	CreatePodsWithControllerRef(namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error
	// DeletePod deletes the pod identified by podID.
	DeletePod(namespace string, podID string, object runtime.Object) error
	// PatchPod patches the pod.
	PatchPod(namespace, name string, data []byte) error
}

// RealPodControl is the default implementation of PodControlInterface.
type RealPodControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

var _ PodControlInterface = &RealPodControl{}

func getPodsLabelSet(template *v1.PodTemplateSpec) labels.Set {
	desiredLabels := make(labels.Set)
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}
	return desiredLabels
}

func getPodsFinalizers(template *v1.PodTemplateSpec) []string {
	desiredFinalizers := make([]string, len(template.Finalizers))
	copy(desiredFinalizers, template.Finalizers)
	return desiredFinalizers
}

func getPodsAnnotationSet(template *v1.PodTemplateSpec) labels.Set {
	desiredAnnotations := make(labels.Set)
	for k, v := range template.Annotations {
		desiredAnnotations[k] = v
	}
	return desiredAnnotations
}

func validatePodName(name string, prefix bool) []string {
	return apimachineryvalidation.NameIsDNSSubdomain(name, prefix)
}

func getPodsPrefix(controllerName string) string {
	// use the dash (if the name isn't too long) to make the pod name a bit prettier
	prefix := fmt.Sprintf("%s-", controllerName)
	if len(validatePodName(prefix, true)) != 0 {
		prefix = controllerName
	}
	return prefix
}

func validateControllerRef(controllerRef *metav1.OwnerReference) error {
	if controllerRef == nil {
		return fmt.Errorf("controllerRef is nil")
	}
	if len(controllerRef.APIVersion) == 0 {
		return fmt.Errorf("controllerRef has empty APIVersion")
	}
	if len(controllerRef.Kind) == 0 {
		return fmt.Errorf("controllerRef has empty Kind")
	}
	if controllerRef.Controller == nil || *controllerRef.Controller != true {
		return fmt.Errorf("controllerRef.Controller is not set to true")
	}
	if controllerRef.BlockOwnerDeletion == nil || *controllerRef.BlockOwnerDeletion != true {
		return fmt.Errorf("controllerRef.BlockOwnerDeletion is not set")
	}
	return nil
}

func (r RealPodControl) CreatePods(namespace string, template *v1.PodTemplateSpec, object runtime.Object) error {
	return r.createPods("", namespace, template, object, nil)
}

func (r RealPodControl) CreatePodsWithControllerRef(namespace string, template *v1.PodTemplateSpec, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createPods("", namespace, template, controllerObject, controllerRef)
}

func (r RealPodControl) CreatePodsOnNode(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createPods(nodeName, namespace, template, object, controllerRef)
}

func (r RealPodControl) PatchPod(namespace, name string, data []byte) error {
	_, err := r.KubeClient.CoreV1().Pods(namespace).Patch(name, types.StrategicMergePatchType, data)
	return err
}

func GetPodFromTemplate(template *v1.PodTemplateSpec, parentObject runtime.Object, controllerRef *metav1.OwnerReference) (*v1.Pod, error) {
	desiredLabels := getPodsLabelSet(template)
	desiredFinalizers := getPodsFinalizers(template)
	desiredAnnotations := getPodsAnnotationSet(template)
	accessor, err := meta.Accessor(parentObject)
	if err != nil {
		return nil, fmt.Errorf("parentObject does not have ObjectMeta, %v", err)
	}
	prefix := getPodsPrefix(accessor.GetName())

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       desiredLabels,
			Annotations:  desiredAnnotations,
			GenerateName: prefix,
			Finalizers:   desiredFinalizers,
		},
	}
	if controllerRef != nil {
		pod.OwnerReferences = append(pod.OwnerReferences, *controllerRef)
	}
	pod.Spec = *template.Spec.DeepCopy()
	return pod, nil
}

func (r RealPodControl) createPods(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	pod, err := GetPodFromTemplate(template, object, controllerRef)
	if err != nil {
		return err
	}
	if len(nodeName) != 0 {
		pod.Spec.NodeName = nodeName
	}
	if labels.Set(pod.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create pods, no labels")
	}
	if newPod, err := r.KubeClient.CoreV1().Pods(namespace).Create(pod); err != nil {
		r.Recorder.Eventf(object, v1.EventTypeWarning, FailedCreatePodReason, "Error creating: %v", err)
		return err
	} else {
		accessor, err := meta.Accessor(object)
		if err != nil {
			glog.Errorf("parentObject does not have ObjectMeta, %v", err)
			return nil
		}
		glog.V(4).Infof("Controller %v created pod %v", accessor.GetName(), newPod.Name)
		r.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreatePodReason, "Created pod: %v", newPod.Name)
	}
	return nil
}

func (r RealPodControl) DeletePod(namespace string, podID string, object runtime.Object) error {
	accessor, err := meta.Accessor(object)
	if err != nil {
		return fmt.Errorf("object does not have ObjectMeta, %v", err)
	}
	glog.V(2).Infof("Controller %v deleting pod %v/%v", accessor.GetName(), namespace, podID)
	if err := r.KubeClient.CoreV1().Pods(namespace).Delete(podID, nil); err != nil {
		r.Recorder.Eventf(object, v1.EventTypeWarning, FailedDeletePodReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete pods: %v", err)
	} else {
		r.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted pod: %v", podID)
	}
	return nil
}

type FakePodControl struct {
	sync.Mutex
	Templates       []v1.PodTemplateSpec
	ControllerRefs  []metav1.OwnerReference
	DeletePodName   []string
	Patches         [][]byte
	Err             error
	CreateLimit     int
	CreateCallCount int
}

var _ PodControlInterface = &FakePodControl{}

func (f *FakePodControl) PatchPod(namespace, name string, data []byte) error {
	f.Lock()
	defer f.Unlock()
	f.Patches = append(f.Patches, data)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakePodControl) CreatePods(namespace string, spec *v1.PodTemplateSpec, object runtime.Object) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("Not creating pod, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *spec)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakePodControl) CreatePodsWithControllerRef(namespace string, spec *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("Not creating pod, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *spec)
	f.ControllerRefs = append(f.ControllerRefs, *controllerRef)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakePodControl) CreatePodsOnNode(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("Not creating pod, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *template)
	f.ControllerRefs = append(f.ControllerRefs, *controllerRef)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakePodControl) DeletePod(namespace string, podID string, object runtime.Object) error {
	f.Lock()
	defer f.Unlock()
	f.DeletePodName = append(f.DeletePodName, podID)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakePodControl) Clear() {
	f.Lock()
	defer f.Unlock()
	f.DeletePodName = []string{}
	f.Templates = []v1.PodTemplateSpec{}
	f.ControllerRefs = []metav1.OwnerReference{}
	f.Patches = [][]byte{}
	f.CreateLimit = 0
	f.CreateCallCount = 0
}

// FilterActivePods returns pods that have not terminated.
func FilterActivePods(pods []*v1.Pod) []*v1.Pod {
	var result []*v1.Pod
	for _, p := range pods {
		if IsPodActive(p) {
			result = append(result, p)
		} else {
			glog.V(4).Infof("Ignoring inactive pod %v/%v in state %v, deletion time %v",
				p.Namespace, p.Name, p.Status.Phase, p.DeletionTimestamp)
		}
	}
	return result
}

func IsPodActive(p *v1.Pod) bool {
	return v1.PodSucceeded != p.Status.Phase &&
		v1.PodFailed != p.Status.Phase &&
		p.DeletionTimestamp == nil
}

// filterRunningPods returns pods that is running.
func filterRunningPods(pods []*v1.Pod) []*v1.Pod {
	var result []*v1.Pod
	for _, p := range pods {
		if isPodRunning(p) {
			result = append(result, p)
		} else {
			glog.V(4).Infof("Ignoring inactive pod %v/%v in state %v, deletion time %v",
				p.Namespace, p.Name, p.Status.Phase, p.DeletionTimestamp)
		}
	}
	return result
}

func isPodRunning(p *v1.Pod) bool {
	return v1.PodRunning == p.Status.Phase
}

// ServiceControlInterface is an interface that knows how to add or delete Services
// created as an interface to allow testing.
type ServiceControlInterface interface {
	// CreateServices creates new Services according to the spec.
	CreateServices(namespace string, service *v1.Service, object runtime.Object) error
	// CreateServicesWithControllerRef creates new services according to the spec, and sets object as the service's controller.
	CreateServicesWithControllerRef(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error
	// PatchService patches the service.
	PatchService(namespace, name string, data []byte) error
}

// RealServiceControl is the default implementation of ServiceControlInterface.
type RealServiceControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

func (r RealServiceControl) PatchService(namespace, name string, data []byte) error {
	_, err := r.KubeClient.CoreV1().Services(namespace).Patch(name, types.StrategicMergePatchType, data)
	return err
}

func (r RealServiceControl) CreateServices(namespace string, service *v1.Service, object runtime.Object) error {
	return r.createServices(namespace, service, object, nil)
}

func (r RealServiceControl) CreateServicesWithControllerRef(namespace string, service *v1.Service, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createServices(namespace, service, controllerObject, controllerRef)
}

func (r RealServiceControl) createServices(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	if labels.Set(service.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create Services, no labels")
	}

	newService, err := r.KubeClient.CoreV1().Services(namespace).Create(service)
	if err != nil {
		r.Recorder.Eventf(object, v1.EventTypeWarning, FailedCreateServiceReason, "Error creating: %v", err)
		return fmt.Errorf("unable to create services: %v", err)
	}

	accessor, err := meta.Accessor(object)
	if err != nil {
		glog.Errorf("parentObject does not have ObjectMeta, %v", err)
		return nil
	}
	glog.V(4).Infof("Controller %v created service %v", accessor.GetName(), newService.Name)
	r.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreateServiceReason, "Created service: %v", newService.Name)

	return nil
}

type FakeServiceControl struct {
	sync.Mutex
	Templates       []v1.Service
	ControllerRefs  []metav1.OwnerReference
	DeletePodName   []string
	Patches         [][]byte
	Err             error
	CreateLimit     int
	CreateCallCount int
}

var _ ServiceControlInterface = &FakeServiceControl{}

func (f *FakeServiceControl) PatchService(namespace, name string, data []byte) error {
	f.Lock()
	defer f.Unlock()
	f.Patches = append(f.Patches, data)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) CreateServices(namespace string, service *v1.Service, object runtime.Object) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("Not creating service, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *service)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) CreateServicesWithControllerRef(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("Not creating service, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *service)
	f.ControllerRefs = append(f.ControllerRefs, *controllerRef)
	if f.Err != nil {
		return f.Err
	}
	return nil
}
