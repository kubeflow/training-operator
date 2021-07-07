package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getDelegatingClientFromClient try to extract client reader from client, client
// reader reads cluster info from api client.
func getDelegatingClientFromClient(c client.Client) (client.Client, error) {
	input := client.NewDelegatingClientInput{
		CacheReader:     nil,
		Client:          c,
		UncachedObjects: nil,
	}
	return client.NewDelegatingClient(input)
}
