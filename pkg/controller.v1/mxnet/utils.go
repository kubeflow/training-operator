package mxnet

import (
	"fmt"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mxv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errPortNotFound = fmt.Errorf("failed to found the port")
)

// getDelegatingClientFromClient try to extract client reader from client, client
// reader reads cluster info from api client.
func getDelegatingClientFromClient(c client.Client) (client.Client, error) {
	input := client.NewDelegatingClientInput{
		CacheReader:     nil,
		Client:          c,
		UncachedObjects: nil,
	}
	return client.NewDelegatingClient(input)
}

// GetPortFromMXJob gets the port of mxnet container.
func GetPortFromMXJob(mxJob *mxv1.MXJob, rtype commonv1.ReplicaType) (int32, error) {
	containers := mxJob.Spec.MXReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == mxv1.DefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == mxv1.DefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, errPortNotFound
}

func ContainSchedulerSpec(mxJob *mxv1.MXJob) bool {
	if _, ok := mxJob.Spec.MXReplicaSpecs[mxv1.MXReplicaTypeScheduler]; ok {
		return true
	}
	return false
}
