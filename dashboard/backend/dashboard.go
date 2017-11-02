package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rs/cors"
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
	fs := http.FileServer(http.Dir("./dashboard/public"))

	mux := http.NewServeMux()
	mux.Handle("/", fs)
	mux.Handle("/api/", apiHandler)
	p := ":8080"
	fmt.Println("Listening on", p)
	corsHandler := cors.Default().Handler(mux)

	http.ListenAndServe(p, corsHandler)
}
