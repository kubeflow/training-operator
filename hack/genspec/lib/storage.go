package lib

import (
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type ResourceInfo struct {
	groupVersionKind schema.GroupVersionKind
	object           runtime.Object
	list             runtime.Object
}

func NewResourceInfo(gvk schema.GroupVersionKind, obj runtime.Object, list runtime.Object) ResourceInfo {
	return ResourceInfo{
		groupVersionKind: gvk,
		object:           obj,
		list:             list,
	}
}

type StandardStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &StandardStorage{}
var _ rest.StandardStorage = &StandardStorage{}

func NewStandardStorage(cfg ResourceInfo) *StandardStorage {
	return &StandardStorage{cfg}
}

func (r *StandardStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.groupVersionKind
}

func (r *StandardStorage) New() runtime.Object {
	return r.cfg.object
}

func (r *StandardStorage) Create(ctx apirequest.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, includeUninitialized bool) (runtime.Object, error) {
	return r.New(), nil
}

func (r *StandardStorage) Get(ctx apirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

func (r *StandardStorage) NewList() runtime.Object {
	return r.cfg.list
}

func (r *StandardStorage) List(ctx apirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

func (r *StandardStorage) Update(ctx apirequest.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (runtime.Object, bool, error) {
	return r.New(), true, nil
}

func (r *StandardStorage) Delete(ctx apirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}

func (r *StandardStorage) DeleteCollection(ctx apirequest.Context, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

func (r *StandardStorage) Watch(ctx apirequest.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return nil, nil
}
