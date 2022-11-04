package util

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestOnDependentXXXFunc(t *testing.T) {
	createfunc := OnDependentCreateFunc(expectation.NewControllerExpectations())
	deletefunc := OnDependentDeleteFunc(expectation.NewControllerExpectations())

	for _, testCase := range []struct {
		object client.Object
		expect bool
	}{
		{
			// pod object with label is allowed
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						commonv1.ReplicaTypeLabel: "Worker",
					},
				},
			},
			expect: true,
		},
		{
			// service object without label is not allowed
			object: &corev1.Service{},
			expect: false,
		},
		{
			// objects other than pod/service are not allowed
			object: &corev1.ConfigMap{},
			expect: false,
		},
	} {
		ret := createfunc(event.CreateEvent{
			Object: testCase.object,
		})
		if ret != testCase.expect {
			t.Errorf("expect %t, but get %t", testCase.expect, ret)
		}
		ret = deletefunc(event.DeleteEvent{
			Object: testCase.object,
		})
		if ret != testCase.expect {
			t.Errorf("expect %t, but get %t", testCase.expect, ret)
		}

	}
}
