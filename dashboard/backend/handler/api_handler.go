// handler is a package handling API requests for managing TfJobs.
// The primary purpose of handler is implementing the functionality needed by the TfJobs dashboard.
package handler

import (
	"fmt"
	"net/http"

	log "github.com/golang/glog"

	"github.com/emicklei/go-restful"
	"github.com/tensorflow/k8s/dashboard/backend/client"
	"github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

// APIHandler handles the API calls
type APIHandler struct {
	cManager client.ClientManager
}

// TfJobDetail describe the specification of a TfJob
// as well as related TensorBoard service if any and related pods
type TfJobDetail struct {
	TfJob     *v1alpha1.TfJob `json:"tfJob"`
	TbService *v1.Service     `json:"tbService"`
	Pods      []v1.Pod        `json:"pods"`
}

// TfJobList is a list of TfJobs
type TfJobList struct {
	tfJobs []v1alpha1.TfJob `json:"TfJobs"`
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
			Reads(v1alpha1.TfJob{}).
			Writes(v1alpha1.TfJob{}))

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
	namespace := "default"
	jobs, err := apiHandler.cManager.TfJobClient.TensorflowV1alpha1().TfJobs(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Warningf("failed to list TfJobs under namespace %v: %v", namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully listed TfJobs under namespace %v", namespace)
		response.WriteHeaderAndEntity(http.StatusOK, jobs)
	}
}

func (apiHandler *APIHandler) handleGetTfJobDetail(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("tfjob")
	job, err := apiHandler.cManager.TfJobClient.TensorflowV1alpha1().TfJobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Infof("cannot find TfJob %v under namespace %v, error: %v", name, namespace, err)
		if errors.IsNotFound(err) {
			response.WriteError(http.StatusNotFound, err)
		} else {
			response.WriteError(http.StatusInternalServerError, err)
		}
		return
	}

	tfJobDetail := TfJobDetail{
		TfJob: job,
	}

	if job.Spec.TensorBoard != nil {
		tbSpec, err := apiHandler.cManager.ClientSet.CoreV1().Services(namespace).List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tensorflow.org=,app=tensorboard,runtime_id=%s", job.Spec.RuntimeId),
		})
		if err != nil {
			log.Warningf("failed to list TensorBoard for TfJob %v under namespace %v, error: %v", job.Name, job.Namespace, err)
			// TODO maybe partial result?
			response.WriteError(http.StatusNotFound, err)
			return
		} else if len(tbSpec.Items) > 0 {
			// Should never be more than 1 service that matched, handle error
			// Handle case where no TensorBoard is found
			tfJobDetail.TbService = &tbSpec.Items[0]
			log.Warningf("more than one TensorBoards found for TfJob %v under namespace %v, this should be impossible",
				job.Name, job.Namespace)
		} else {
			log.Warningf("Couldn't find a TensorBoard service for TfJob %v under namespace %v", job.Name, job.Namespace)
		}
	}

	// Get associated pods
	pods, err := apiHandler.cManager.ClientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tensorflow.org=,runtime_id=%s", job.Spec.RuntimeId),
	})
	if err != nil {
		log.Warningf("failed to list pods for TfJob %v under namespace %v: %v", name, namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully listed pods for TfJob %v under namespace %v", name, namespace)
		tfJobDetail.Pods = pods.Items
		response.WriteHeaderAndEntity(http.StatusOK, tfJobDetail)
	}
}

func (apiHandler *APIHandler) handleDeploy(request *restful.Request, response *restful.Response) {
	clt := apiHandler.cManager.TfJobClient
	tfJob := new(v1alpha1.TfJob)
	if err := request.ReadEntity(tfJob); err != nil {
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	j, err := clt.TensorflowV1alpha1().TfJobs(tfJob.Namespace).Create(tfJob)
	if err != nil {
		log.Warningf("failed to deploy TfJob %v under namespace %v: %v", tfJob.Name, tfJob.Namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully deployed TfJob %v under namespace %v", tfJob.Name, tfJob.Namespace)
		response.WriteHeaderAndEntity(http.StatusCreated, j)
	}
}

func (apiHandler *APIHandler) handleDeleteTfJob(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("tfjob")
	clt := apiHandler.cManager.TfJobClient
	err := clt.TensorflowV1alpha1().TfJobs(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		log.Warningf("failed to delete TfJob %v under namespace %v: %v", name, namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully deleted TfJob %v under namespace %v", name, namespace)
		response.WriteHeader(http.StatusNoContent)
	}
}

func (apiHandler *APIHandler) handleGetPodLogs(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("podname")
	logs, err := apiHandler.cManager.ClientSet.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{}).Do().Raw()
	if err != nil {
		log.Warningf("failed to get pod logs for TfJob %v under namespace %v: %v", name, namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully get pod logs for TfJob %v under namespace %v", name, namespace)
		response.WriteHeaderAndEntity(http.StatusOK, string(logs))
	}
}
