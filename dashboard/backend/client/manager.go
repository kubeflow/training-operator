package client

import (
	"github.com/tensorflow/k8s/pkg/util/k8sutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	CRDGroup   = "mlkube.io"
	CRDVersion = "v1beta1"
)

type ClientManager struct {
	restCfg     *rest.Config
	ClientSet   *kubernetes.Clientset
	TfJobClient *k8sutil.TfJobRestClient
}

func (c *ClientManager) init() {
	restCfg, err := k8sutil.InClusterConfig()
	if err != nil {
		panic(err)
	}
	c.restCfg = restCfg

	clientset, err := kubernetes.NewForConfig(c.restCfg)
	if err != nil {
		panic(err.Error())
	}
	c.ClientSet = clientset

	tfJobClient, err := k8sutil.NewTfJobClient()
	if err != nil {
		panic(err)
	}
	c.TfJobClient = tfJobClient
}

func NewClientManager() (ClientManager, error) {
	cm := ClientManager{}
	cm.init()

	return cm, nil
}
