// client is a package handling authentication/communication with kubernetes API
package client

import (
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/util/k8sutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ClientManager is responsible for initializing and creating clients to communicate with
// kubernetes apiserver on demand
type ClientManager struct {
	restCfg     *rest.Config
	ClientSet   *kubernetes.Clientset
	TFJobClient *versioned.Clientset
}

// init methods initiates the TFJob client for given cluster config
func (c *ClientManager) init() {
	restCfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		panic(err)
	}
	c.restCfg = restCfg

	clientset, err := kubernetes.NewForConfig(c.restCfg)
	if err != nil {
		panic(err.Error())
	}
	c.ClientSet = clientset

	tfJobClient := versioned.NewForConfigOrDie(c.restCfg)

	c.TFJobClient = tfJobClient
}

// NewClientManager creates and init a new instance of ClientManager
func NewClientManager() (ClientManager, error) {
	cm := ClientManager{}
	cm.init()

	return cm, nil
}
