package test_job

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DummyClient struct {
	scheme *runtime.Scheme
	mapper meta.RESTMapper
	client.Reader
	client.Writer
	client.StatusClient
	Cache []client.Object
}

func (c *DummyClient) Scheme() *runtime.Scheme {
	return c.scheme
}

func (c *DummyClient) RESTMapper() meta.RESTMapper {
	return c.mapper
}

func (c *DummyClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.Cache = append(c.Cache, obj)
	return nil
}

func (c *DummyClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	for idx, o := range c.Cache {
		if o.GetName() == obj.GetName() && o.GetNamespace() == obj.GetNamespace() && o.GetObjectKind() == obj.GetObjectKind() {
			c.Cache = append(c.Cache[:idx], c.Cache[idx+1:]...)
			return nil
		}
	}
	return errors.NewNotFound(schema.GroupResource{
		Group:    obj.GetObjectKind().GroupVersionKind().Group,
		Resource: obj.GetSelfLink(),
	}, obj.GetName())
}

func (c *DummyClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	for idx, o := range c.Cache {
		if o.GetName() == obj.GetName() && o.GetNamespace() == obj.GetNamespace() && o.GetObjectKind() == obj.GetObjectKind() {
			c.Cache[idx] = obj
			return nil
		}
	}
	return errors.NewNotFound(schema.GroupResource{
		Group:    obj.GetObjectKind().GroupVersionKind().Group,
		Resource: obj.GetSelfLink(),
	}, obj.GetName())
}
