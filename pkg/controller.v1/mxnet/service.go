package mxnet

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *MXJobReconciler) GetServicesForJob(job interface{}) ([]*corev1.Service, error) {
	mxJob, err := meta.Accessor(job)
	if err != nil {
		return nil, fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	// List all services to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList, client.MatchingLabels(r.GenLabels(mxJob.GetName())))
	if err != nil {
		return nil, err
	}
	return convertServiceList(serviceList.Items), nil
}

// convertServiceList convert service list to service point list
func convertServiceList(list []corev1.Service) []*corev1.Service {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Service, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}
