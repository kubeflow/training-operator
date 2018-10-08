package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/kubeflow/tf-operator/dashboard/backend/client"
	"github.com/kubeflow/tf-operator/dashboard/backend/handler"
)

const (
	DefaultFrontendDir string = "/opt/tensorflow_k8s/dashboard/frontend/build/"
	DefaultBackendPort int    = 8080
)

func main() {
	var frontendDir string
	var port int
	flag.StringVar(&frontendDir, "frontend-dir", DefaultFrontendDir,
		`directory of the dashboard frontend`)
	flag.IntVar(&port, "port", DefaultBackendPort,
		`port this program will listen`)
	flag.Parse()
	log.SetOutput(os.Stdout)
	cm, err := client.NewClientManager()
	if err != nil {
		log.Fatalf("Error while initializing connection to Kubernetes apiserver: %v", err)
	}
	apiHandler, err := handler.CreateHTTPAPIHandler(cm)
	if err != nil {
		log.Fatalf("Error while creating the API Handler: %v", err)
	}

	http.Handle("/tfjobs/ui/", http.StripPrefix("/tfjobs/ui/", http.FileServer(http.Dir(frontendDir))))
	http.Handle("/tfjobs/api/", apiHandler)

	p := ":" + strconv.Itoa(port)
	log.Println("Dashboard available at /tfjobs/ui/ on port", p)

	if err = http.ListenAndServe(p, nil); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
