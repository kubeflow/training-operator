package handler

import (
	"fmt"
	"net/http"

	"github.com/tensorflow/k8s/dashboard/backend/resources/tfjob"

	restful "github.com/emicklei/go-restful"
	"github.com/tensorflow/k8s/dashboard/backend/client"
)

type APIHandler struct {
	cManager client.ClientManager
}

func CreateHTTPAPIHandler(client client.ClientManager) (http.Handler, error) {
	apiHandler := APIHandler{
		cManager: client,
	}
	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)

	apiV1Ws := new(restful.WebService)

	apiV1Ws.Path("/api").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	wsContainer.Add(apiV1Ws)

	apiV1Ws.Route(
		apiV1Ws.GET("/tfjob").
			To(apiHandler.handleGetTFJobs).
			Writes(tfjob.TFJobList{}))

	return wsContainer, nil
}

func (apiHandler *APIHandler) handleGetTFJobs(request *restful.Request, response *restful.Response) {
	fmt.Println("listing tfjobs")

	jobs, err := apiHandler.cManager.TfJobClient.List("default")

	if err != nil {
		panic(err)
	}

	response.WriteHeaderAndEntity(http.StatusOK, jobs)
}
