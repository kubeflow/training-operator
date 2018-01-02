// handler is a package handling API requests for managing TfJobs.
// The primary purpose of handler is implementing the functionality needed by the TfJobs dashboard.
package handler

import (
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful"
	"github.com/tensorflow/k8s/dashboard/backend/client"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tfv1alpha1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"

)

// APIHandler handles the API calls
type APIHandler struct {
	cManager client.ClientManager
}

// TfJobDetail describe the specification of a TfJob
// as well as related TensorBoard service if any and related pods
type TfJobDetail struct {
	TfJob     *tfv1alpha1.TfJob `json:"tfJob"`
	TbService *v1.Service `json:"tbService"`
	Pods      []v1.Pod    `json:"pods"`
}

// TfJobList is a list of TfJobs
type TfJobList struct {
	tfJobs []tfv1alpha1.TfJob `json:"TfJobs"`
}

// CreateHTTPAPIHandler creates the restful Container and defines the routes the API will serve
func CreateHTTPAPIHandler(client client.ClientManager) (http.Handler, error) {
	apiHandler := APIHandler{
		cManager: client,
	}

	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)

	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "DELETE"},
		CookiesAllowed: false,
		Container:      wsContainer,
	}
	wsContainer.Filter(cors.Filter)
	wsContainer.Filter(wsContainer.OPTIONSFilter)

	apiV1Ws := new(restful.WebService)

	apiV1Ws.Path("/api").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

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
			Reads(tfv1alpha1.TfJob{}).
			Writes(tfv1alpha1.TfJob{}))

	apiV1Ws.Route(
		apiV1Ws.DELETE("/tfjob/{namespace}/{tfjob}").
			To(apiHandler.handleDeleteTfJob))

	apiV1Ws.Route(
		apiV1Ws.GET("/logs/{namespace}/{podname}").
			To(apiHandler.handleGetPodLogs).
			Writes([]byte{}))

	wsContainer.Add(apiV1Ws)
	return wsContainer, nil
}

func (apiHandler *APIHandler) handleGetTfJobs(request *restful.Request, response *restful.Response) {

	//TODO: namespace handling
	jobs, err := apiHandler.cManager.TfJobClient.TensorflowV1alpha1().TfJobs("default").List(metav1.ListOptions{})

	if err != nil {
		panic(err)
	}

	response.WriteHeaderAndEntity(http.StatusOK, jobs)
}

func (apiHandler *APIHandler) handleGetTfJobDetail(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("tfjob")

	job, err := apiHandler.cManager.TfJobClient.TensorflowV1alpha1().TfJobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	tfJobDetail := TfJobDetail{
		TfJob: job,
	}

	if job.Spec.TensorBoard != nil {
		tbSpec, err := apiHandler.cManager.ClientSet.CoreV1().Services(namespace).List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tensorflow.org=,app=tensorboard,runtime_id=%s", job.Spec.RuntimeId),
		})
		if err != nil {
			panic(err)
		}

		if len(tbSpec.Items) > 0 {
			// Should never be more than 1 service that matched, handle error
			// Handle case where no tensorboard is found
			tfJobDetail.TbService = &tbSpec.Items[0]
		} else {
			fmt.Println(fmt.Sprintf("Couldn't find a TensorBoard service for TfJob %s", job.ObjectMeta.Name))
		}
	}

	// Get associated pods
	pods, err := apiHandler.cManager.ClientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tensorflow.org=,runtime_id=%s", job.Spec.RuntimeId),
	})
	if err != nil {
		panic(err)
	}
	tfJobDetail.Pods = pods.Items

	response.WriteHeaderAndEntity(http.StatusOK, tfJobDetail)
}

func (apiHandler *APIHandler) handleDeploy(request *restful.Request, response *restful.Response) {
	client := apiHandler.cManager.TfJobClient
	spec := new(tfv1alpha1.TfJob)
	if err := request.ReadEntity(spec); err != nil {
		panic(err)
	}
	j, err := client.TensorflowV1alpha1().TfJobs(spec.ObjectMeta.Namespace).Create(spec)
	if err != nil {
		panic(err)
	}
	response.WriteHeaderAndEntity(http.StatusCreated, j)
}

func (apiHandler *APIHandler) handleDeleteTfJob(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("tfjob")
	client := apiHandler.cManager.TfJobClient
	err := client.TensorflowV1alpha1().TfJobs(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		panic(err)
	}
	response.WriteHeader(http.StatusOK)
}

func (apiHandler *APIHandler) handleGetPodLogs(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("podname")

	logs, err := apiHandler.cManager.ClientSet.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{}).Do().Raw()
	if err != nil {
		panic(err)
	}

	response.WriteHeaderAndEntity(http.StatusOK, string(logs))
}
