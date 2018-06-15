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

package controller

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/controller"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
)

func TestAddTFJob(t *testing.T) {
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
	ctr, _, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
	ctr.tfJobInformerSynced = alwaysReady
	ctr.podInformerSynced = alwaysReady
	ctr.serviceInformerSynced = alwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(threadCount, stopCh)
	}
	go run(stopCh)

	var key string
	syncChan := make(chan string)
	ctr.syncHandler = func(tfJobKey string) (bool, error) {
		key = tfJobKey
		<-syncChan
		return true, nil
	}
	ctr.updateStatusHandler = func(tfjob *tfv1alpha2.TFJob) error {
		return nil
	}

	tfJob := newTFJob(1, 0)
	unstructured, err := convertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}
	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}
	ctr.addTFJob(unstructured)

	syncChan <- "sync"
	if key != getKey(tfJob, t) {
		t.Errorf("Failed to enqueue the TFJob %s: expected %s, got %s", tfJob.Name, getKey(tfJob, t), key)
	}
	close(stopCh)
}

func TestCopyLabelsAndAnnotation(t *testing.T) {
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
	ctr, _, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
	fakePodControl := &controller.FakePodControl{}
	ctr.podControl = fakePodControl
	ctr.tfJobInformerSynced = alwaysReady
	ctr.podInformerSynced = alwaysReady
	ctr.serviceInformerSynced = alwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(threadCount, stopCh)
	}
	go run(stopCh)

	ctr.updateStatusHandler = func(tfJob *tfv1alpha2.TFJob) error {
		return nil
	}

	tfJob := newTFJob(1, 0)
	annotations := map[string]string{
		"annotation1": "1",
	}
	labels := map[string]string{
		"label1": "1",
	}
	tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].Template.Labels = labels
	tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].Template.Annotations = annotations
	unstructured, err := convertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}

	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}

	_, err = ctr.syncTFJob(getKey(tfJob, t))
	if err != nil {
		t.Errorf("%s: unexpected error when syncing jobs %v", tfJob.Name, err)
	}

	if len(fakePodControl.Templates) != 1 {
		t.Errorf("Expected to create 1 pod while got %d", len(fakePodControl.Templates))
	}
	actual := fakePodControl.Templates[0]
	v, exist := actual.Labels["label1"]
	if !exist {
		t.Errorf("Labels does not exist")
	}
	if v != "1" {
		t.Errorf("Labels value do not equal")
	}

	v, exist = actual.Annotations["annotation1"]
	if !exist {
		t.Errorf("Annotations does not exist")
	}
	if v != "1" {
		t.Errorf("Annotations value does not equal")
	}

	close(stopCh)
}

func newTFJobWithChief(worker, ps int) *tfv1alpha2.TFJob {
	tfJob := newTFJob(worker, ps)
	tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeChief] = &tfv1alpha2.TFReplicaSpec{
		Template: newTFReplicaSpecTemplate(),
	}
	return tfJob
}

func newTFJob(worker, ps int) *tfv1alpha2.TFJob {
	tfJob := &tfv1alpha2.TFJob{
		TypeMeta: metav1.TypeMeta{
			Kind: tfv1alpha2.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testTFJobName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: tfv1alpha2.TFJobSpec{
			TFReplicaSpecs: make(map[tfv1alpha2.TFReplicaType]*tfv1alpha2.TFReplicaSpec),
		},
	}

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &tfv1alpha2.TFReplicaSpec{
			Replicas: &worker,
			Template: newTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker] = workerReplicaSpec
	}

	if ps > 0 {
		ps := int32(ps)
		psReplicaSpec := &tfv1alpha2.TFReplicaSpec{
			Replicas: &ps,
			Template: newTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypePS] = psReplicaSpec
	}
	return tfJob
}

func getKey(tfJob *tfv1alpha2.TFJob, t *testing.T) string {
	key, err := KeyFunc(tfJob)
	if err != nil {
		t.Errorf("Unexpected error getting key for job %v: %v", tfJob.Name, err)
		return ""
	}
	return key
}

func checkCondition(tfJob *tfv1alpha2.TFJob, condition tfv1alpha2.TFJobConditionType, reason string) bool {
	for _, v := range tfJob.Status.Conditions {
		if v.Type == condition && v.Status == v1.ConditionTrue && v.Reason == reason {
			return true
		}
	}
	return false
}

func newTFReplicaSpecTemplate() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  tfv1alpha2.DefaultContainerName,
					Image: testImageName,
					Args:  []string{"Fake", "Fake"},
					Ports: []v1.ContainerPort{
						v1.ContainerPort{
							Name:          tfv1alpha2.DefaultPortName,
							ContainerPort: tfv1alpha2.DefaultPort,
						},
					},
				},
			},
		},
	}
}
