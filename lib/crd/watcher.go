/*
Copyright 2022. projectsveltos.io. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package crd

import (
	"context"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
)

type ChangeType string

const (
	Add    ChangeType = "add"
	Delete ChangeType = "delete"
	Modify ChangeType = "modify"
)

type handler func(gvk *schema.GroupVersionKind, action ChangeType)

// WatchCustomResourceDefinition starts a watcher for CustomResourceDefinition.
// When new CRD is added/deleted/modified, invokes the passed handler.
// Caller must have RBAC to watch CustomResourceDefinition.
//
// This intentionally does not keep a local cache of CustomResourceDefinitions:
// the handler only ever needs the GVK carried by the event that triggered it,
// so there is nothing to gain (and memory to lose) by retaining every CRD in
// an indexed store the way a SharedIndexInformer would.
func WatchCustomResourceDefinition(ctx context.Context, config *rest.Config,
	h handler, logger logr.Logger) {

	gvk := schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}

	lw, err := getCRDListerWatcher(ctx, &gvk, config)
	if err != nil {
		logger.Error(err, "Failed to get lister watcher")
		return
	}

	runCRDReflector(ctx, lw, h, logger)
}

func getCRDListerWatcher(ctx context.Context, gvk *schema.GroupVersionKind, config *rest.Config,
) (cache.ListerWatcher, error) {

	d, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dc := discovery.NewDiscoveryClientForConfigOrDie(config)
	groupResources, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		// getCRDListerWatcher is only called after verifying resource
		// is installed.
		return nil, err
	}

	resourceId := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: mapping.Resource.Resource,
	}

	// CustomResourceDefinition is cluster-scoped, so no namespace is set.
	resourceClient := d.Resource(resourceId)

	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return resourceClient.List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return resourceClient.Watch(ctx, options)
		},
	}, nil
}

// runCRDReflector keeps CustomResourceDefinitions list/watched, using client-go's
// Reflector for the resilient list-then-watch/relist-on-error behavior, but backed
// by a store that discards every object right after notifying the handler, instead
// of an Indexer that would keep all of them in memory for the lifetime of the process.
func runCRDReflector(ctx context.Context, lw cache.ListerWatcher, h handler, logger logr.Logger) {
	store := &eventOnlyStore{handler: h, logger: logger}
	reflector := cache.NewReflector(lw, &unstructured.Unstructured{}, store, 0)
	reflector.Run(ctx.Done())
}

// eventOnlyStore implements cache.Store but never retains any object: every
// Add/Update/Delete (including the ones replayed from the initial List) is
// translated into a GVK and forwarded to the handler, then dropped.
type eventOnlyStore struct {
	handler handler
	logger  logr.Logger
}

func (s *eventOnlyStore) Add(obj interface{}) error {
	s.notify(obj, Add)
	return nil
}

func (s *eventOnlyStore) Update(obj interface{}) error {
	s.notify(obj, Modify)
	return nil
}

func (s *eventOnlyStore) Delete(obj interface{}) error {
	s.notify(obj, Delete)
	return nil
}

// Replace is invoked by the Reflector after each (re)list, once per listed object.
// Those are treated the same way an informer's initial sync would: as Add events.
func (s *eventOnlyStore) Replace(list []interface{}, _ string) error {
	for i := range list {
		s.notify(list[i], Add)
	}
	return nil
}

func (s *eventOnlyStore) Resync() error {
	return nil
}

func (s *eventOnlyStore) List() []interface{} {
	return nil
}

func (s *eventOnlyStore) ListKeys() []string {
	return nil
}

func (s *eventOnlyStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *eventOnlyStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *eventOnlyStore) notify(obj interface{}, action ChangeType) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), crd)
	if err != nil {
		s.logger.Error(err, "could not convert obj to CustomResourceDefinition")
		return
	}

	for i := range crd.Spec.Versions {
		gvk := &schema.GroupVersionKind{
			Group:   crd.Spec.Group,
			Version: crd.Spec.Versions[i].Name,
			Kind:    crd.Spec.Names.Kind,
		}
		s.handler(gvk, action)
	}
}
