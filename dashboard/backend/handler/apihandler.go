package handler

import (
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful"
	"github.com/tensorflow/k8s/dashboard/backend/client"
	"github.com/tensorflow/k8s/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type APIHandler struct {
	cManager client.ClientManager
}

type StorageDetails struct{}

//add tensorboard ips, add azure file / gs links
type TfJobDetail struct {
	TfJob          *spec.TfJob `json:"tfJob"`
	TbService      *v1.Service `json:"tbService"`
	StorageDetails StorageDetails
}

type TfJobList struct {
	TfJobs []spec.TfJob `json:"tfjobs"`
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
			To(apiHandler.handleGetTfJobs).
			Writes(TfJobList{}))

	apiV1Ws.Route(
		apiV1Ws.GET("/tfjob/{namespace}/{tfjob}").
			To(apiHandler.handleGetTfJobDetail).
			Writes(TfJobDetail{}))

	apiV1Ws.Route(
		apiV1Ws.POST("/tfjob").
			To(apiHandler.handleDeploy).
			Reads(spec.TfJob{}).
			Writes(spec.TfJob{}))

	return wsContainer, nil
}

func (apiHandler *APIHandler) handleGetTfJobs(request *restful.Request, response *restful.Response) {

	//TODO: namespace handling
	jobs, err := apiHandler.cManager.TfJobClient.List("default")

	if err != nil {
		panic(err)
	}

	response.WriteHeaderAndEntity(http.StatusOK, jobs)
}

func (apiHandler *APIHandler) handleGetTfJobDetail(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("tfjob")

	job, err := apiHandler.cManager.TfJobClient.Get(namespace, name)
	if err != nil {
		panic(err)
	}

	tfJobDetail := TfJobDetail{
		TfJob: job,
	}

	if job.Spec.TensorBoard != nil {
		tbSpec, err := apiHandler.cManager.ClientSet.CoreV1().Services("default").List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=tensorboard,runtime_id=%s", job.Spec.RuntimeId),
		})
		if err != nil {
			panic(err)
		}

		// Should never be more than 1 service that matched, handle error
		// Handle case where no tensorboard is found
		tfJobDetail.TbService = &tbSpec.Items[0]
	}

	response.WriteHeaderAndEntity(http.StatusOK, tfJobDetail)
}

func (apiHandler *APIHandler) handleDeploy(request *restful.Request, response *restful.Response) {
	client := apiHandler.cManager.TfJobClient
	spec := new(spec.TfJob)
	if err := request.ReadEntity(spec); err != nil {
		panic(err)
	}
	j, err := client.Create(spec.Metadata.Namespace, spec)
	if err != nil {
		panic(err)
	}
	response.WriteHeaderAndEntity(http.StatusCreated, j)
}
