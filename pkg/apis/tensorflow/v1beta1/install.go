package v1beta1

import (
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Install registers the API group and adds type to a schema
func Install(groupFactoryRegister announced.APIGroupFactoryRegistry, registry *registered.APIRegistrationManager, scheme *runtime.Scheme) error {
	return announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:              GroupName,
			RootScopedKinds:        sets.NewString(),
			VersionPreferenceOrder: []string{SchemeGroupVersion.Version},
		},
		announced.VersionToSchemeFunc{
			SchemeGroupVersion.Version: AddToScheme,
		},
	).Announce(groupFactoryRegister).RegisterAndEnable(registry, scheme)
}
