/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package xgboost

import (
	"fmt"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	xgboostv1 "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"
)

// TODO (Jeffwan@): Find an elegant way to either use delegatingReader or directly use clientss

// getClientReaderFromClient try to extract client reader from client, client
// reader reads cluster info from api client.
func getClientReaderFromClient(c client.Client) (client.Reader, error) {
	//if dr, err := getDelegatingReader(c); err != nil {
	//	return nil, err
	//} else {
	//	return dr.ClientReader, nil
	//}

	//return dr, nil

	return nil, nil
}

// getDelegatingReader try to extract DelegatingReader from client.
//func getDelegatingReader(c client.Client) (*client.DelegatingReader, error) {
//	dc, ok := c.(*client.DelegatingClient)
//	if !ok {
//		return nil, errors.New("cannot convert from Client to DelegatingClient")
//	}
//	dr, ok := dc.Reader.(*client.DelegatingReader)
//	if !ok {
//		return nil, errors.New("cannot convert from DelegatingClient.Reader to Delegating Reader")
//	}
//	return dr, nil
//}

func computeMasterAddr(jobName, rtype, index string) string {
	n := jobName + "-" + rtype + "-" + index
	return strings.Replace(n, "/", "-", -1)
}

// GetPortFromXGBoostJob gets the port of xgboost container.
func GetPortFromXGBoostJob(job *xgboostv1.XGBoostJob, rtype xgboostv1.XGBoostJobReplicaType) (int32, error) {
	containers := job.Spec.XGBReplicaSpecs[commonv1.ReplicaType(rtype)].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == xgboostv1.DefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == xgboostv1.DefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("failed to found the port")
}

func computeTotalReplicas(obj metav1.Object) int32 {
	job := obj.(*xgboostv1.XGBoostJob)
	jobReplicas := int32(0)

	if job.Spec.XGBReplicaSpecs == nil || len(job.Spec.XGBReplicaSpecs) == 0 {
		return jobReplicas
	}
	for _, r := range job.Spec.XGBReplicaSpecs {
		if r.Replicas == nil {
			continue
		} else {
			jobReplicas += *r.Replicas
		}
	}
	return jobReplicas
}

func createClientSets(config *restclientset.Config) (kubeclientset.Interface, kubeclientset.Interface, volcanoclient.Interface, error) {
	if config == nil {
		println("there is an error for the input config")
		return nil, nil, nil, nil
	}

	kubeClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "xgboostjob-operator"))
	if err != nil {
		return nil, nil, nil, err
	}

	leaderElectionClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "leader-election"))
	if err != nil {
		return nil, nil, nil, err
	}

	volcanoClientSet, err := volcanoclient.NewForConfig(restclientset.AddUserAgent(config, "volcano"))
	if err != nil {
		return nil, nil, nil, err
	}

	return kubeClientSet, leaderElectionClientSet, volcanoClientSet, nil
}
