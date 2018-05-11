package main

import (
	"log"
	"net/http"
	"os"

	"github.com/kubeflow/tf-operator/dashboard/backend/client"
	"github.com/kubeflow/tf-operator/dashboard/backend/handler"
)

func main() {
	log.SetOutput(os.Stdout)
	cm, err := client.NewClientManager()
	if err != nil {
		log.Fatalf("Error while initializing connection to Kubernetes apiserver: %v", err)
	}
	apiHandler, err := handler.CreateHTTPAPIHandler(cm)
	if err != nil {
		log.Fatalf("Error while creating the API Handler: %v", err)
	}

	http.Handle("/tfjobs/ui/", http.StripPrefix("/tfjobs/ui/", http.FileServer(http.Dir("/opt/tensorflow_k8s/dashboard/frontend/build/"))))
	http.Handle("/tfjobs/api/", apiHandler)

	p := ":8080"
	log.Println("Dashboard available at /tfjobs/ui/ on port", p)

	if err = http.ListenAndServe(p, nil); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
