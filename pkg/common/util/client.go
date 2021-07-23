package util

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// TODO (Jeffwan@): Find an elegant way to either use delegatingReader or directly use clientss
// GetDelegatingClientFromClient try to extract client reader from client, client
// reader reads cluster info from api client.
func GetDelegatingClientFromClient(c client.Client) (client.Client, error) {
	input := client.NewDelegatingClientInput{
		CacheReader:     nil,
		Client:          c,
		UncachedObjects: nil,
	}
	return client.NewDelegatingClient(input)
}
