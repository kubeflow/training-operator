package common

import (
	"testing"

	"github.com/kubeflow/training-operator/pkg/core"

	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCalculateServiceSliceSize(t *testing.T) {
	type testCase struct {
		services     []*corev1.Service
		replicas     int
		expectedSize int
	}

	services := []*corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "0"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "1"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "2"},
			},
		},
	}

	var testCases = []testCase{
		{
			services:     services,
			replicas:     3,
			expectedSize: 3,
		},
		{
			services:     services,
			replicas:     4,
			expectedSize: 4,
		},
		{
			services:     services,
			replicas:     2,
			expectedSize: 3,
		},
		{
			services: append(services, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{apiv1.ReplicaIndexLabel: "4"},
				},
			}),
			replicas:     3,
			expectedSize: 5,
		},
	}

	for _, tc := range testCases {
		result := core.CalculateServiceSliceSize(tc.services, tc.replicas)
		assert.Equal(t, tc.expectedSize, result)
	}
}

func TestFilterServicesForReplicaType(t *testing.T) {
	services := []*v1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "a",
				Labels: map[string]string{apiv1.ReplicaTypeLabel: "foo"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "b",
				Labels: map[string]string{apiv1.ReplicaTypeLabel: "bar"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "c",
				Labels: map[string]string{apiv1.ReplicaTypeLabel: "foo"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "d",
				Labels: map[string]string{apiv1.ReplicaTypeLabel: "bar"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "e",
				Labels: map[string]string{
					apiv1.ReplicaTypeLabel: "foo",
				},
			},
		},
	}
	c := &JobController{}
	got, err := c.FilterServicesForReplicaType(services, "foo")
	if err != nil {
		t.Fatalf("FilterPodsForReplicaType returned error: %v", err)
	}
	want := []*v1.Service{services[0], services[2], services[4]}
	assert.Equal(t, want, got)
}
