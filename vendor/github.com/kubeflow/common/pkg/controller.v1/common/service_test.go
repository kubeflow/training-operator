package common

import (
	"testing"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
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
		result := calculateServiceSliceSize(tc.services, tc.replicas)
		assert.Equal(t, tc.expectedSize, result)
	}
}
