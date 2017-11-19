package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tensorflow/k8s/dashboard/backend/client"
	"github.com/tensorflow/k8s/dashboard/backend/handler"
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

	http.Handle("/api/", apiHandler)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("/opt/mlkube/dashboard/frontend/build/"))))
	p := ":8081"
	fmt.Println("Listening on", p)

	http.ListenAndServe(p, nil)

}
