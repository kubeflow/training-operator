package tfjob

import "k8s.io/client-go/kubernetes"

type TFJobList struct {
	TFJobs []TFJob `json:"tfjobs"`
}

type TFJob struct {
	Name string
}

func GetTfJobList(c *kubernetes.Clientset) {
	c.CoreV1().RESTClient().Get().RequestURI("listTfJobsURI(ns)").DoRaw()
}
