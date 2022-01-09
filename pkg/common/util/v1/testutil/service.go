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

package testutil

import (
	"context"
	"fmt"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DummyPortName string = "dummy"
	DummyPort     int32  = 1221
)

func NewBaseService(name string, job metav1.Object, refs []metav1.OwnerReference) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          map[string]string{},
			Namespace:       job.GetNamespace(),
			OwnerReferences: refs,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: DummyPortName,
					Port: DummyPort,
				},
			},
		},
	}
}

func NewService(job metav1.Object, typ string, index int, refs []metav1.OwnerReference) *corev1.Service {
	svc := NewBaseService(fmt.Sprintf("%s-%s-%d", job.GetName(), typ, index), job, refs)
	svc.Labels[commonv1.ReplicaTypeLabelDeprecated] = typ
	svc.Labels[commonv1.ReplicaTypeLabel] = typ
	svc.Labels[commonv1.ReplicaIndexLabelDeprecated] = fmt.Sprintf("%d", index)
	svc.Labels[commonv1.ReplicaIndexLabel] = fmt.Sprintf("%d", index)
	return svc
}

// NewServiceList creates count pods with the given phase for the given tfJob
func NewServiceList(count int32, job metav1.Object, typ string, refs []metav1.OwnerReference) []*corev1.Service {
	services := []*corev1.Service{}
	for i := int32(0); i < count; i++ {
		newService := NewService(job, typ, int(i), refs)
		services = append(services, newService)
	}
	return services
}

func SetServices(client client.Client, job metav1.Object, typ string, activeWorkerServices int32,
	refs []metav1.OwnerReference, basicLabels map[string]string) {
	ctx := context.Background()
	for _, svc := range NewServiceList(activeWorkerServices, job, typ, refs) {
		for k, v := range basicLabels {
			svc.Labels[k] = v
		}
		err := client.Create(ctx, svc)
		if errors.IsAlreadyExists(err) {
			return
		} else {
			Expect(err).To(BeNil())
		}
	}
}
