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
	"reflect"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/golang/protobuf/proto"
	"github.com/kubeflow/tf-operator/cmd/tf-operator.v1beta1/app/options"
	common "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta1"
	tfv1beta1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta1"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	"github.com/kubeflow/tf-operator/pkg/common/util/testutil"
	"github.com/kubeflow/tf-operator/pkg/control"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	tfJobRunning   = common.JobRunning
	tfJobSucceeded = common.JobSucceeded
)

func newTFController(
	config *rest.Config,
	kubeClientSet kubeclientset.Interface,
	tfJobClientSet tfjobclientset.Interface,
	resyncPeriod controller.ResyncPeriodFunc,
	option options.ServerOption,
) (
	*TFController,
	kubeinformers.SharedInformerFactory, tfjobinformers.SharedInformerFactory,
) {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientSet, resyncPeriod())
	tfJobInformerFactory := tfjobinformers.NewSharedInformerFactory(tfJobClientSet, resyncPeriod())

	tfJobInformer := NewUnstructuredTFJobInformer(config, metav1.NamespaceAll)

	ctr := NewTFController(tfJobInformer, kubeClientSet, tfJobClientSet, kubeInformerFactory, tfJobInformerFactory, option)
	ctr.PodControl = &controller.FakePodControl{}
	ctr.ServiceControl = &control.FakeServiceControl{}
	return ctr, kubeInformerFactory, tfJobInformerFactory
}

func TestNormalPath(t *testing.T) {
	testCases := map[string]struct {
		worker int
		ps     int

		// pod setup
		ControllerError error
		jobKeyForget    bool

		pendingWorkerPods   int32
		activeWorkerPods    int32
		succeededWorkerPods int32
		failedWorkerPods    int32

		pendingPSPods   int32
		activePSPods    int32
		succeededPSPods int32
		failedPSPods    int32

		activeWorkerServices int32
		activePSServices     int32

		// expectations
		expectedPodCreations     int32
		expectedPodDeletions     int32
		expectedServiceCreations int32

		expectedActiveWorkerPods    int32
		expectedSucceededWorkerPods int32
		expectedFailedWorkerPods    int32

		expectedActivePSPods    int32
		expectedSucceededPSPods int32
		expectedFailedPSPods    int32

		expectedCondition       *common.JobConditionType
		expectedConditionReason string

		// There are some cases that should not check start time since the field should be set in the previous sync loop.
		needCheckStartTime bool
	}{
		"Local TFJob is created": {
			1, 0,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0,
			1, 0, 1,
			0, 0, 0,
			0, 0, 0,
			// We can not check if it is created since the condition is set in addTFJob.
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created": {
			4, 2,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0,
			6, 0, 6,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created and all replicas are pending": {
			4, 2,
			nil, true,
			4, 0, 0, 0,
			2, 0, 0, 0,
			4, 2,
			0, 0, 0,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created and all replicas are running": {
			4, 2,
			nil, true,
			0, 4, 0, 0,
			0, 2, 0, 0,
			4, 2,
			0, 0, 0,
			4, 0, 0,
			2, 0, 0,
			&tfJobRunning, tfJobRunningReason,
			true,
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending": {
			4, 2,
			nil, true,
			2, 0, 0, 0,
			1, 0, 0, 0,
			2, 1,
			3, 0, 3,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is running": {
			4, 2,
			nil, true,
			2, 1, 0, 0,
			1, 0, 0, 0,
			3, 1,
			2, 0, 2,
			1, 0, 0,
			0, 0, 0,
			&tfJobRunning, tfJobRunningReason,
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is succeeded": {
			4, 2,
			nil, true,
			2, 0, 1, 0,
			1, 0, 0, 0,
			3, 1,
			2, 0, 2,
			0, 1, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is succeeded": {
			4, 2,
			nil, true,
			0, 0, 4, 0,
			0, 0, 2, 0,
			4, 2,
			0, 0, 0,
			0, 4, 0,
			0, 2, 0,
			&tfJobSucceeded, tfJobSucceededReason,
			false,
		},
	}

	for name, tc := range testCases {
		// Prepare the clientset and controller for the test.
		kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &v1.SchemeGroupVersion,
			},
		},
		)
		config := &rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &tfv1beta1.SchemeGroupVersion,
			},
		}
		option := options.ServerOption{}
		tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, option)
		ctr.tfJobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		tfJobIndexer := ctr.tfJobInformer.GetIndexer()

		var actual *tfv1beta1.TFJob
		ctr.updateStatusHandler = func(tfJob *tfv1beta1.TFJob) error {
			actual = tfJob
			return nil
		}

		// Run the test logic.
		tfJob := testutil.NewTFJob(tc.worker, tc.ps)
		unstructured, err := testutil.ConvertTFJobToUnstructured(tfJob)
		if err != nil {
			t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
		}

		if err := tfJobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, tfJob, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, t)
		testutil.SetPodsStatuses(podIndexer, tfJob, testutil.LabelPS, tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
		testutil.SetServices(serviceIndexer, tfJob, testutil.LabelWorker, tc.activeWorkerServices, t)
		testutil.SetServices(serviceIndexer, tfJob, testutil.LabelPS, tc.activePSServices, t)

		forget, err := ctr.syncTFJob(testutil.GetKey(tfJob, t))
		// We need requeue syncJob task if podController error
		if tc.ControllerError != nil {
			if err == nil {
				t.Errorf("%s: Syncing jobs would return error when podController exception", name)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected error when syncing jobs %v", name, err)
			}
		}
		if forget != tc.jobKeyForget {
			t.Errorf("%s: unexpected forget value. Expected %v, saw %v\n", name, tc.jobKeyForget, forget)
		}

		fakePodControl := ctr.PodControl.(*controller.FakePodControl)
		fakeServiceControl := ctr.ServiceControl.(*control.FakeServiceControl)
		if int32(len(fakePodControl.Templates)) != tc.expectedPodCreations {
			t.Errorf("%s: unexpected number of pod creates.  Expected %d, saw %d\n", name, tc.expectedPodCreations, len(fakePodControl.Templates))
		}
		if int32(len(fakeServiceControl.Templates)) != tc.expectedServiceCreations {
			t.Errorf("%s: unexpected number of service creates.  Expected %d, saw %d\n", name, tc.expectedServiceCreations, len(fakeServiceControl.Templates))
		}
		if int32(len(fakePodControl.DeletePodName)) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", name, tc.expectedPodDeletions, len(fakePodControl.DeletePodName))
		}
		// Each create should have an accompanying ControllerRef.
		if len(fakePodControl.ControllerRefs) != int(tc.expectedPodCreations) {
			t.Errorf("%s: unexpected number of ControllerRefs.  Expected %d, saw %d\n", name, tc.expectedPodCreations, len(fakePodControl.ControllerRefs))
		}
		// Make sure the ControllerRefs are correct.
		for _, controllerRef := range fakePodControl.ControllerRefs {
			if got, want := controllerRef.APIVersion, tfv1beta1.SchemeGroupVersion.String(); got != want {
				t.Errorf("controllerRef.APIVersion = %q, want %q", got, want)
			}
			if got, want := controllerRef.Kind, tfv1beta1.Kind; got != want {
				t.Errorf("controllerRef.Kind = %q, want %q", got, want)
			}
			if got, want := controllerRef.Name, tfJob.Name; got != want {
				t.Errorf("controllerRef.Name = %q, want %q", got, want)
			}
			if got, want := controllerRef.UID, tfJob.UID; got != want {
				t.Errorf("controllerRef.UID = %q, want %q", got, want)
			}
			if controllerRef.Controller == nil || !*controllerRef.Controller {
				t.Errorf("controllerRef.Controller is not set to true")
			}
		}
		// Validate worker status.
		if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypeWorker)] != nil {
			if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypeWorker)].Active != tc.expectedActiveWorkerPods {
				t.Errorf("%s: unexpected number of active pods.  Expected %d, saw %d\n",
					name, tc.expectedActiveWorkerPods,
					actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypeWorker)].Active)
			}
			if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypeWorker)].Succeeded != tc.expectedSucceededWorkerPods {
				t.Errorf("%s: unexpected number of succeeded pods.  Expected %d, saw %d\n",
					name, tc.expectedSucceededWorkerPods,
					actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypeWorker)].Succeeded)
			}
			if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypeWorker)].Failed != tc.expectedFailedWorkerPods {
				t.Errorf("%s: unexpected number of failed pods.  Expected %d, saw %d\n",
					name, tc.expectedFailedWorkerPods,
					actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypeWorker)].Failed)
			}
		}
		// Validate PS status.
		if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypePS)] != nil {
			if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypePS)].Active != tc.expectedActivePSPods {
				t.Errorf("%s: unexpected number of active pods.  Expected %d, saw %d\n",
					name, tc.expectedActivePSPods,
					actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypePS)].Active)
			}
			if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypePS)].Succeeded != tc.expectedSucceededPSPods {
				t.Errorf("%s: unexpected number of succeeded pods.  Expected %d, saw %d\n",
					name, tc.expectedSucceededPSPods,
					actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypePS)].Succeeded)
			}
			if actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypePS)].Failed != tc.expectedFailedPSPods {
				t.Errorf("%s: unexpected number of failed pods.  Expected %d, saw %d\n",
					name, tc.expectedFailedPSPods,
					actual.Status.ReplicaStatuses[common.ReplicaType(tfv1beta1.TFReplicaTypePS)].Failed)
			}
		}
		// Validate StartTime.
		if tc.needCheckStartTime && actual.Status.StartTime == nil {
			t.Errorf("%s: StartTime was not set", name)
		}
		// Validate conditions.
		if tc.expectedCondition != nil && !testutil.CheckCondition(actual, *tc.expectedCondition, tc.expectedConditionReason) {
			t.Errorf("%s: expected condition %#v, got %#v", name, *tc.expectedCondition, actual.Status.Conditions)
		}
	}
}

func TestRun(t *testing.T) {
	// Prepare the clientset and controller for the test.
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)
	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &tfv1beta1.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady

	stopCh := make(chan struct{})
	go func() {
		// It is a hack to let the controller stop to run without errors.
		// We can not just send a struct to stopCh because there are multiple
		// receivers in controller.Run.
		time.Sleep(testutil.SleepInterval)
		stopCh <- struct{}{}
	}()
	err := ctr.Run(testutil.ThreadCount, stopCh)
	if err != nil {
		t.Errorf("Failed to run: %v", err)
	}
}

func TestSyncPdb(t *testing.T) {
	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &tfv1beta1.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	kubeClientSet := fake.NewSimpleClientset()
	option := options.ServerOption{
		EnableGangScheduling: true,
	}
	ctr, _, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, option)

	type testCase struct {
		tfJob     *tfv1beta1.TFJob
		expectPdb *v1beta1.PodDisruptionBudget
	}

	minAvailable2 := intstr.FromInt(2)
	testCases := []testCase{
		{
			tfJob: &tfv1beta1.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sync-pdb",
				},
				Spec: tfv1beta1.TFJobSpec{
					TFReplicaSpecs: map[tfv1beta1.TFReplicaType]*common.ReplicaSpec{
						tfv1beta1.TFReplicaTypeWorker: &common.ReplicaSpec{
							Replicas: proto.Int32(1),
						},
					},
				},
			},
			expectPdb: nil,
		},
		{
			tfJob: &tfv1beta1.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sync-pdb",
				},
				Spec: tfv1beta1.TFJobSpec{
					TFReplicaSpecs: map[tfv1beta1.TFReplicaType]*common.ReplicaSpec{
						tfv1beta1.TFReplicaTypeWorker: &common.ReplicaSpec{
							Replicas: proto.Int32(2),
						},
					},
				},
			},
			expectPdb: &v1beta1.PodDisruptionBudget{
				Spec: v1beta1.PodDisruptionBudgetSpec{
					MinAvailable: &minAvailable2,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"tf_job_name": "test-sync-pdb",
						},
					},
				},
			},
		},
	}
	for _, c := range testCases {
		pdb, _ := ctr.SyncPdb(c.tfJob, getTotalReplicas(c.tfJob))
		if pdb == nil && c.expectPdb != nil {
			t.Errorf("Got nil, want %v", c.expectPdb.Spec)
		}

		if pdb != nil && !reflect.DeepEqual(c.expectPdb.Spec, pdb.Spec) {
			t.Errorf("Got %+v, want %+v", pdb.Spec, c.expectPdb.Spec)
		}
	}
}
