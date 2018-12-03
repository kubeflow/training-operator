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
	"os"
	"testing"

	"k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/kubeflow/tf-operator/cmd/tf-operator.v2/app/options"
	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/util/testutil"
)

func TestAddPod(t *testing.T) {
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
			GroupVersion: &tfv1alpha2.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	var key string
	syncChan := make(chan string)
	ctr.syncHandler = func(tfJobKey string) (bool, error) {
		key = tfJobKey
		<-syncChan
		return true, nil
	}

	tfJob := testutil.NewTFJob(1, 0)
	unstructured, err := testutil.ConvertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}

	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}
	pod := testutil.NewPod(tfJob, testutil.LabelWorker, 0, t)
	ctr.AddPod(pod)

	syncChan <- "sync"
	if key != testutil.GetKey(tfJob, t) {
		t.Errorf("Failed to enqueue the TFJob %s: expected %s, got %s", tfJob.Name, testutil.GetKey(tfJob, t), key)
	}
	close(stopCh)
}

func TestClusterSpec(t *testing.T) {
	type tc struct {
		tfJob               *tfv1alpha2.TFJob
		rt                  string
		index               string
		customClusterDomain string
		expectedClusterSpec string
	}
	testCase := []tc{
		tc{
			tfJob:               testutil.NewTFJobWithNamespace(1, 0, "ns0"),
			rt:                  "worker",
			index:               "0",
			customClusterDomain: "",
			expectedClusterSpec: `{"cluster":{"worker":["` + testutil.TestTFJobName +
				`-worker-0.ns0.svc:2222"]},"task":{"type":"worker","index":0},"environment":"cloud"}`,
		},
		tc{
			tfJob:               testutil.NewTFJobWithNamespace(1, 0, "ns1"),
			rt:                  "worker",
			index:               "0",
			customClusterDomain: "tf.training.com",
			expectedClusterSpec: `{"cluster":{"worker":["` + testutil.TestTFJobName +
				`-worker-0.ns1.svc.tf.training.com:2222"]},"task":{"type":"worker","index":0},"environment":"cloud"}`,
		},
		tc{
			tfJob:               testutil.NewTFJobWithNamespace(1, 1, "ns2"),
			rt:                  "worker",
			index:               "0",
			customClusterDomain: "tf.training.org",
			expectedClusterSpec: `{"cluster":{"ps":["` + testutil.TestTFJobName +
				`-ps-0.ns2.svc.tf.training.org:2222"],"worker":["` + testutil.TestTFJobName +
				`-worker-0.ns2.svc.tf.training.org:2222"]},"task":{"type":"worker","index":0},"environment":"cloud"}`,
		},
		tc{
			tfJob:               testutil.NewTFJobWithEvaluatorAndNamespace(1, 1, 1, "ns3"),
			rt:                  "worker",
			index:               "0",
			customClusterDomain: "tf.training.io",
			expectedClusterSpec: `{"cluster":{"ps":["` + testutil.TestTFJobName +
				`-ps-0.ns3.svc.tf.training.io:2222"],"worker":["` + testutil.TestTFJobName +
				`-worker-0.ns3.svc.tf.training.io:2222"]},"task":{"type":"worker","index":0},"environment":"cloud"}`,
		},
	}
	for _, c := range testCase {
		os.Setenv(EnvCustomClusterDomain, c.customClusterDomain)
		demoTemplateSpec := c.tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].Template
		if err := setClusterSpec(&demoTemplateSpec, c.tfJob, c.rt, c.index); err != nil {
			t.Errorf("Failed to set cluster spec: %v", err)
		}
		actual := demoTemplateSpec.Spec.Containers[0].Env[0].Value
		if c.expectedClusterSpec != actual {
			t.Errorf("Expected %s, got %s", c.expectedClusterSpec, actual)
		}
	}
}

func TestRestartPolicy(t *testing.T) {
	type tc struct {
		tfJob                 *tfv1alpha2.TFJob
		expectedRestartPolicy v1.RestartPolicy
		expectedType          tfv1alpha2.TFReplicaType
	}
	testCase := []tc{
		func() tc {
			tfJob := testutil.NewTFJob(1, 0)
			specRestartPolicy := tfv1alpha2.RestartPolicyExitCode
			tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				tfJob: tfJob,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          tfv1alpha2.TFReplicaTypeWorker,
			}
		}(),
		func() tc {
			tfJob := testutil.NewTFJob(1, 0)
			specRestartPolicy := tfv1alpha2.RestartPolicyNever
			tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				tfJob: tfJob,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          tfv1alpha2.TFReplicaTypeWorker,
			}
		}(),
		func() tc {
			tfJob := testutil.NewTFJob(1, 0)
			specRestartPolicy := tfv1alpha2.RestartPolicyAlways
			tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				tfJob: tfJob,
				expectedRestartPolicy: v1.RestartPolicyAlways,
				expectedType:          tfv1alpha2.TFReplicaTypeWorker,
			}
		}(),
		func() tc {
			tfJob := testutil.NewTFJob(1, 0)
			specRestartPolicy := tfv1alpha2.RestartPolicyOnFailure
			tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].RestartPolicy = specRestartPolicy
			return tc{
				tfJob: tfJob,
				expectedRestartPolicy: v1.RestartPolicyOnFailure,
				expectedType:          tfv1alpha2.TFReplicaTypeWorker,
			}
		}(),
	}
	for _, c := range testCase {
		spec := c.tfJob.Spec.TFReplicaSpecs[c.expectedType]
		podTemplate := spec.Template
		setRestartPolicy(&podTemplate, spec)
		if podTemplate.Spec.RestartPolicy != c.expectedRestartPolicy {
			t.Errorf("Expected %s, got %s", c.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
		}
	}
}

func TestExitCode(t *testing.T) {
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
			GroupVersion: &tfv1alpha2.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, kubeInformerFactory, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	fakePodControl := &controller.FakePodControl{}
	ctr.PodControl = fakePodControl
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()
	podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	ctr.updateStatusHandler = func(tfJob *tfv1alpha2.TFJob) error {
		return nil
	}

	tfJob := testutil.NewTFJob(1, 0)
	tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].RestartPolicy = tfv1alpha2.RestartPolicyExitCode
	unstructured, err := testutil.ConvertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}

	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}
	pod := testutil.NewPod(tfJob, testutil.LabelWorker, 0, t)
	pod.Status.Phase = v1.PodFailed
	pod.Spec.Containers = append(pod.Spec.Containers, v1.Container{})
	pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, v1.ContainerStatus{
		Name: tfv1alpha2.DefaultContainerName,
		State: v1.ContainerState{
			Terminated: &v1.ContainerStateTerminated{
				ExitCode: 130,
			},
		},
	})

	if err := podIndexer.Add(pod); err != nil {
		t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
	}
	_, err = ctr.syncTFJob(testutil.GetKey(tfJob, t))
	if err != nil {
		t.Errorf("%s: unexpected error when syncing jobs %v", tfJob.Name, err)
	}

	found := false
	for _, deletedPodName := range fakePodControl.DeletePodName {
		if deletedPodName == pod.Name {
			found = true
		}
	}
	if !found {
		t.Errorf("Failed to delete pod %s", pod.Name)
	}
	close(stopCh)
}
