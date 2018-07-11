package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	"github.com/kubeflow/tf-operator/hack/genspec/lib"
	"github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
)

type Config struct {
	Registry *registered.APIRegistrationManager
	Scheme   *runtime.Scheme
	Codecs   serializer.CodecFactory

	Info              spec.InfoProps
	OpenAPIDefinition common.GetOpenAPIDefinitions
}

// newServerConfigWithSpec returns a serverConfig with OpenAPI config.
// This serverConfig will be used to set up a temporary server to generate specification.
func newServerConfigWithSpec(cfg Config) (*genericapiserver.RecommendedConfig, error) {
	recommendedOptions := genericoptions.NewRecommendedOptions("/registry/foo.com", cfg.Codecs.LegacyCodec())
	recommendedOptions.SecureServing.BindPort = 8443
	recommendedOptions.Etcd = nil
	recommendedOptions.Authentication = nil
	recommendedOptions.Authorization = nil
	recommendedOptions.CoreAPI = nil

	if err := recommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(cfg.Codecs)
	if err := recommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(cfg.OpenAPIDefinition, cfg.Scheme)
	serverConfig.OpenAPIConfig.Info.InfoProps = cfg.Info

	return serverConfig, nil
}

// routeStorage returns a storage for RESTful services, which contains CRUD operation to CRD TFJob.
func routeStorage(groupVersionResource schema.GroupVersionResource, registry *registered.APIRegistrationManager, schema *runtime.Scheme) (map[string]rest.Storage, error) {
	mapper := registry.RESTMapper()

	groupVersionKind, err := mapper.KindFor(groupVersionResource)
	if err != nil {
		return nil, err
	}

	object, err := schema.New(groupVersionKind)
	if err != nil {
		return nil, err
	}

	list, err := schema.New(groupVersionKind.GroupVersion().WithKind(groupVersionKind.Kind + "List"))
	if err != nil {
		return nil, err
	}

	storage := lib.NewStandardStorage(lib.NewResourceInfo(groupVersionKind, object, list))

	return map[string]rest.Storage{groupVersionResource.Resource: storage}, nil
}

// generateSwaggerJson generates OpenAPI specification swagger.json with generated model
// pkg/apis/tensorflow/v1alpha2/openapi_generated.go and API routing information.
func generateSwaggerJson() (string, error) {
	var (
		groupFactoryRegistry = make(announced.APIGroupFactoryRegistry)
		registry             = registered.NewOrDie("")
		Scheme               = runtime.NewScheme()
		Codecs               = serializer.NewCodecFactory(Scheme)
	)

	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// Register the API group to schema
	if err := v1alpha2.Install(groupFactoryRegistry, registry, Scheme); err != nil {
		return "", fmt.Errorf("failed to register API group to schema: %v", err)
	}

	serverConfig, err := newServerConfigWithSpec(
		Config{
			Registry: registry,
			Scheme:   Scheme,
			Codecs:   Codecs,
			Info: spec.InfoProps{
				Version: "v1alpha2",
				Contact: &spec.ContactInfo{
					Name: "kubeflow.org",
					URL:  "https://kubeflow.org",
				},
				License: &spec.License{
					Name: "Apache 2.0",
					URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
				},
			},
			OpenAPIDefinition: v1alpha2.GetOpenAPIDefinitions,
		})
	if err != nil {
		return "", fmt.Errorf("failed to create server config: %v", err)
	}

	genericServer, err := serverConfig.Complete().New("openapi-server", genericapiserver.EmptyDelegate)
	if err != nil {
		return "", fmt.Errorf("failed to create server: %v", err)
	}

	// Add routing information to this server
	groupVersionResource := v1alpha2.SchemeGroupVersion.WithResource(v1alpha2.Plural)
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(groupVersionResource.Group, registry, Scheme, metav1.ParameterCodec, Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = groupVersionResource.GroupVersion()
	storage, err := routeStorage(groupVersionResource, registry, Scheme)
	if err != nil {
		return "", err
	}

	apiGroupInfo.VersionedResourcesStorageMap[groupVersionResource.Version] = storage
	if err := genericServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return "", err
	}

	spec, err := builder.BuildOpenAPISpec(genericServer.Handler.GoRestfulContainer.RegisteredWebServices(), serverConfig.OpenAPIConfig)
	if err != nil {
		return "", fmt.Errorf("failed to render OpenAPI spec: %v", err)
	}
	data, err := json.MarshalIndent(spec, "", "	")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func main() {
	filename := flag.String("f", "pkg/apis/tensorflow/v1alpha2/openapi-spec/swagger.json", "Path to write OpenAPI spec file")

	flag.Parse()

	err := os.MkdirAll(filepath.Dir(*filename), 0644)
	if err != nil {
		glog.Fatalf("failed to create directory %s: %v", filepath.Dir(*filename), err)
	}

	apiSpec, err := generateSwaggerJson()
	if err != nil {
		glog.Fatalf("failed to generate spec: %v", err)
	}

	err = ioutil.WriteFile(*filename, []byte(apiSpec), 0644)
	if err != nil {
		glog.Fatalf("failed to write spec: %v", err)
	}
}
