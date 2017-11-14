package trainer

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"reflect"
	"sync"

	"github.com/tensorflow/k8s/pkg/spec"
	tfJobFake "github.com/tensorflow/k8s/pkg/util/k8sutil/fake"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"github.com/tensorflow/k8s/pkg/util"
)

func TestTBReplicaSet(t *testing.T) {
	clientSet := fake.NewSimpleClientset()

	jobSpec := &spec.TfJob{
		Metadata: meta_v1.ObjectMeta {
			Name: "some-job",
			UID: "some-uid",
		},
		Spec: spec.TfJobSpec{
			RuntimeId: "some-runtime",
			ReplicaSpecs: []*spec.TfReplicaSpec{
				{
					Replicas:      proto.Int32(1),
					TfPort:        proto.Int32(10),
					Template:      &v1.PodTemplateSpec{},
					TfReplicaType: spec.MASTER,
				},
			},
			TensorBoard: &spec.TensorBoardSpec{
				LogDir: "/tmp/tensorflow",
			},
		},
	}

	stopC := make(chan struct{})

	wg := &sync.WaitGroup{}
	job, err := initJob(clientSet, &tfJobFake.TfJobClientFake{}, jobSpec, stopC, wg)

	if err != nil {
		t.Fatalf("initJob failed: %v", err)
	}

	replica, err := NewTBReplicaSet(clientSet, *jobSpec.Spec.TensorBoard, job)

	if err != nil {
		t.Fatalf("NewTBReplicaSet failed: %v", err)
	}

	if err := replica.Create(); err != nil {
		t.Fatalf("TBReplicaSet.Create() error; %v", err)
	}

	// Expected labels
	expectedLabels := map[string]string{
		"tensorflow.org":  "",
		"app":        "tensorboard",
		"runtime_id": "some-runtime",
		"tf_job_name": "some-job",
	}

	trueVal := true
	expectedOwnerReference := meta_v1.OwnerReference{
		APIVersion: "",
		Kind: "",
		Name: "some-job",
		UID: "some-uid",
		Controller: &trueVal,
		BlockOwnerDeletion: &trueVal,
	}

	// Check that a service was created.
	// TODO: Change this List for a Get for clarity
	sList, err := clientSet.CoreV1().Services(replica.Job.job.Metadata.Namespace).List(meta_v1.ListOptions{})
	if err != nil {
		t.Fatalf("List services error; %v", err)
	}

	if len(sList.Items) != 1 {
		t.Fatalf("Expected 1 service got %v", len(sList.Items))
	}

	s := sList.Items[0]

	if !reflect.DeepEqual(expectedLabels, s.ObjectMeta.Labels) {
		t.Fatalf("Service Labels; Got %v Want: %v", s.ObjectMeta.Labels, expectedLabels)
	}

	name := "some-job-tensorboard-some-runtime"
	if s.ObjectMeta.Name != name {
		t.Fatalf("Job.ObjectMeta.Name = %v; want %v", s.ObjectMeta.Name, name)
	}

	if len(s.ObjectMeta.OwnerReferences) != 1 {
		t.Fatalf("Expected 1 owner reference got %v", len(s.ObjectMeta.OwnerReferences))
	}

	if !reflect.DeepEqual(s.ObjectMeta.OwnerReferences[0], expectedOwnerReference)  {
		t.Fatalf("Service.Metadata.OwnerReferences; Got %v; want %v", util.Pformat(s.ObjectMeta.OwnerReferences[0]), util.Pformat(expectedOwnerReference))
	}

	// Check that a deployment was created.
	l, err := clientSet.ExtensionsV1beta1().Deployments(replica.Job.job.Metadata.Namespace).List(meta_v1.ListOptions{})
	if err != nil {
		t.Fatalf("List deployments error; %v", err)
	}

	if len(l.Items) != 1 {
		t.Fatalf("Expected 1 deployment got %v", len(l.Items))
	}

	d := l.Items[0]

	if !reflect.DeepEqual(expectedLabels, d.ObjectMeta.Labels) {
		t.Fatalf("Deployment Labels; Got %v Want: %v", expectedLabels, d.ObjectMeta.Labels)
	}

	if d.ObjectMeta.Name != name {
		t.Fatalf("Deployment.ObjectMeta.Name = %v; want %v", d.ObjectMeta.Name, name)
	}

	if len(d.ObjectMeta.OwnerReferences) != 1 {
		t.Fatalf("Expected 1 owner reference got %v", len(d.ObjectMeta.OwnerReferences))
	}

	if !reflect.DeepEqual(d.ObjectMeta.OwnerReferences[0], expectedOwnerReference)  {
		t.Fatalf("Service.Metadata.OwnerReferences; Got %v; want %v", util.Pformat(s.ObjectMeta.OwnerReferences[0]), util.Pformat(expectedOwnerReference))
	}

	// Delete the job.
	// N.B it doesn't look like the Fake clientset is sophisticated enough to delete jobs in response to a
	// DeleteCollection request (deleting individual jobs does appear to work with the Fake). So if we were to list
	// the jobs after calling Delete we'd still see the job. So we will rely on E2E tests to verify Delete works
	// correctly.
	if err := replica.Delete(); err != nil {
		t.Fatalf("TBReplicaSet.Delete() error; %v", err)
	}
}
