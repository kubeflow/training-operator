//Package handler is a package handling API requests for managing TFJobs.
// The primary purpose of handler is implementing the functionality needed by the TFJobs dashboard.
package handler

import (
	"fmt"
	"net/http"

	log "github.com/golang/glog"

	"github.com/emicklei/go-restful"
	"github.com/kubeflow/tf-operator/dashboard/backend/client"
	"github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha1"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

// APIHandler handles the API calls
type APIHandler struct {
	cManager client.ClientManager
}

// TFJobDetail describe the specification of a TFJob
// if any and related pods
type TFJobDetail struct {
	TFJob *v1alpha1.TFJob `json:"tfJob"`
	Pods  []v1.Pod        `json:"pods"`
}

// TFJobList is a list of TFJobs
type TFJobList struct {
	TFJobs []v1alpha1.TFJob `json:"TFJobs"`
}

// NamespaceList is a list of namespaces
type NamespaceList struct {
	Namespaces []v1.Namespace `json:"Namespaces"`
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

	// Issue was figuring out which part of the url corresponded to the application itself, and which parts were related to the proxying.
	// If we have this url with kubectl proxy + ambassador:
	// `127.0.0.1:8001/api/v1/namespaces/kubeflow/services/ambassador:80/proxy/tfjobs/ui/`
	// Proxy parts are:
	// `127.0.0.1:8001/api/v1/namespaces/kubeflow/services/ambassador:80/proxy`
	// Application parts are (depending on how the rewrite is configured in ambassador):
	// `/tfjobs/ui/`
	// So the way the application handles it, is finding the first occurence of tfjobs in the url and appending /api/ to reach the backend.
	// Finally, by rewriting in ambassador to `/tfjobs/` and not `/` and by having the backend listening on
	// `/tfjobs/ui` and `/tfjobs/api` we ensure that the logic won't break when not using ambassador, i.e:
	// When using a dev server: `127.0.0.1:3000/tfjobs/ui`
	// When proxying directly on the dashboard service:
	// `http://127.0.0.1:8001/api/v1/namespaces/kubeflow/services/tf-job-dashboard:80/proxy/tfjobs/ui/#/`
	apiV1Ws.Path("/tfjobs/api").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	apiV1Ws.Route(
		apiV1Ws.GET("/tfjob").
			To(apiHandler.handleGetTFJobs).
			Writes(TFJobList{}))

	apiV1Ws.Route(
		apiV1Ws.GET("/tfjob/{namespace}").
			To(apiHandler.handleGetTFJobs).
			Writes(TFJobList{}))

	apiV1Ws.Route(
		apiV1Ws.GET("/tfjob/{namespace}/{tfjob}").
			To(apiHandler.handleGetTFJobDetail).
			Writes(TFJobDetail{}))

	apiV1Ws.Route(
		apiV1Ws.POST("/tfjob").
			To(apiHandler.handleDeploy).
			Reads(v1alpha1.TFJob{}).
			Writes(v1alpha1.TFJob{}))

	apiV1Ws.Route(
		apiV1Ws.DELETE("/tfjob/{namespace}/{tfjob}").
			To(apiHandler.handleDeleteTFJob))

	apiV1Ws.Route(
		apiV1Ws.GET("/logs/{namespace}/{podname}").
			To(apiHandler.handleGetPodLogs).
			Writes([]byte{}))

	apiV1Ws.Route(
		apiV1Ws.GET("/namespace").
			To(apiHandler.handleGetNamespaces).
			Writes(NamespaceList{}))

	wsContainer.Add(apiV1Ws)
	return wsContainer, nil
}

func (apiHandler *APIHandler) handleGetTFJobs(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	jobs, err := apiHandler.cManager.TFJobClient.KubeflowV1alpha1().TFJobs(namespace).List(metav1.ListOptions{})

	ns := "all"
	if namespace != "" {
		ns = namespace
	}
	if err != nil {
		log.Warningf("failed to list TFJobs under %v namespace(s): %v", ns, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully listed TFJobs under %v namespace(s)", ns)
		response.WriteHeaderAndEntity(http.StatusOK, jobs)
	}
}

func (apiHandler *APIHandler) handleGetTFJobDetail(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("tfjob")
	job, err := apiHandler.cManager.TFJobClient.KubeflowV1alpha1().TFJobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Infof("cannot find TFJob %v under namespace %v, error: %v", name, namespace, err)
		if errors.IsNotFound(err) {
			response.WriteError(http.StatusNotFound, err)
		} else {
			response.WriteError(http.StatusInternalServerError, err)
		}
		return
	}

	tfJobDetail := TFJobDetail{
		TFJob: job,
	}

	// Get associated pods
	pods, err := apiHandler.cManager.ClientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubeflow.org=,runtime_id=%s", job.Spec.RuntimeId),
	})
	if err != nil {
		log.Warningf("failed to list pods for TFJob %v under namespace %v: %v", name, namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully listed pods for TFJob %v under namespace %v", name, namespace)
		tfJobDetail.Pods = pods.Items
		response.WriteHeaderAndEntity(http.StatusOK, tfJobDetail)
	}
}

func (apiHandler *APIHandler) handleDeploy(request *restful.Request, response *restful.Response) {
	clt := apiHandler.cManager.TFJobClient
	tfJob := new(v1alpha1.TFJob)
	if err := request.ReadEntity(tfJob); err != nil {
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	_, err := apiHandler.cManager.ClientSet.CoreV1().Namespaces().Get(tfJob.Namespace, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		// If namespace doesn't exist we create it
		_, nsErr := apiHandler.cManager.ClientSet.CoreV1().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tfJob.Namespace}})
		if nsErr != nil {
			log.Warningf("failed to create namespace %v for TFJob %v: %v", tfJob.Namespace, tfJob.Name, nsErr)
			response.WriteError(http.StatusInternalServerError, nsErr)
		}
	} else if err != nil {
		log.Warningf("failed to deploy TFJob %v under namespace %v: %v", tfJob.Name, tfJob.Namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	}

	j, err := clt.KubeflowV1alpha1().TFJobs(tfJob.Namespace).Create(tfJob)
	if err != nil {
		log.Warningf("failed to deploy TFJob %v under namespace %v: %v", tfJob.Name, tfJob.Namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully deployed TFJob %v under namespace %v", tfJob.Name, tfJob.Namespace)
		response.WriteHeaderAndEntity(http.StatusCreated, j)
	}
}

func (apiHandler *APIHandler) handleDeleteTFJob(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("tfjob")
	clt := apiHandler.cManager.TFJobClient
	err := clt.KubeflowV1alpha1().TFJobs(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		log.Warningf("failed to delete TFJob %v under namespace %v: %v", name, namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully deleted TFJob %v under namespace %v", name, namespace)
		response.WriteHeader(http.StatusNoContent)
	}
}

func (apiHandler *APIHandler) handleGetPodLogs(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("podname")
	logs, err := apiHandler.cManager.ClientSet.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{}).Do().Raw()
	if err != nil {
		log.Warningf("failed to get pod logs for TFJob %v under namespace %v: %v", name, namespace, err)
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("successfully get pod logs for TFJob %v under namespace %v", name, namespace)
		response.WriteHeaderAndEntity(http.StatusOK, string(logs))
	}
}

func (apiHandler *APIHandler) handleGetNamespaces(request *restful.Request, response *restful.Response) {
	l, err := apiHandler.cManager.ClientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Warningf("failed to list namespaces.")
		response.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Infof("sucessfully listed namespaces")
		response.WriteHeaderAndEntity(http.StatusOK, l)
	}
}
