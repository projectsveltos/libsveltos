/*
Copyright 2025. projectsveltos.io. All rights reserved.

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

package pullmode

import (
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	libsveltosset "github.com/projectsveltos/libsveltos/lib/set"
)

type stagedManager struct {
	mu             sync.RWMutex
	stagedBundles  map[string]*libsveltosset.Set
	currentBundles map[corev1.ObjectReference]*libsveltosv1beta1.ConfigurationBundle
}

var (
	instance *stagedManager
	once     sync.Once
)

func getStagedResourcesManager() *stagedManager {
	once.Do(func() {
		instance = &stagedManager{
			stagedBundles:  make(map[string]*libsveltosset.Set),
			currentBundles: make(map[corev1.ObjectReference]*libsveltosv1beta1.ConfigurationBundle),
		}
	})
	return instance
}

func (s *stagedManager) geKey(clusterNamespace, clusterName, requestorName, requestorFeature string) string {
	return fmt.Sprintf("%s/%s/%s/%s", clusterNamespace, clusterName, requestorName, requestorFeature)
}

func (s *stagedManager) storeBundle(clusterNamespace, clusterName, requestorName, requestorFeature string,
	bundle *libsveltosv1beta1.ConfigurationBundle) {

	s.mu.Lock()
	defer s.mu.Unlock()
	key := s.geKey(clusterNamespace, clusterName, requestorName, requestorFeature)
	if _, ok := s.stagedBundles[key]; !ok {
		s.stagedBundles[key] = &libsveltosset.Set{}
	}
	s.stagedBundles[key].Insert(&corev1.ObjectReference{Namespace: bundle.Namespace, Name: bundle.Name})
	s.currentBundles[corev1.ObjectReference{Namespace: bundle.Namespace, Name: bundle.Name}] = bundle
}

func (s *stagedManager) getBundles(clusterNamespace, clusterName, requestorName, requestorFeature string,
) []libsveltosv1beta1.ConfigurationBundle {

	s.mu.RLock()
	defer s.mu.RUnlock()
	key := s.geKey(clusterNamespace, clusterName, requestorName, requestorFeature)

	v, ok := s.stagedBundles[key]
	if !ok {
		return []libsveltosv1beta1.ConfigurationBundle{}
	}

	items := v.Items()
	bundles := make([]libsveltosv1beta1.ConfigurationBundle, v.Len())
	for i := range items {
		ref := corev1.ObjectReference{Namespace: items[i].Namespace, Name: items[i].Name}
		bundles[i] = *s.currentBundles[ref]
	}

	return bundles
}

func (s *stagedManager) clearBundles(clusterNamespace, clusterName, requestorName, requestorFeature string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := s.geKey(clusterNamespace, clusterName, requestorName, requestorFeature)

	bundles, ok := s.stagedBundles[key]
	if !ok {
		return
	}
	items := bundles.Items()
	for i := range items {
		ref := corev1.ObjectReference{Namespace: items[i].Namespace, Name: items[i].Name}
		delete(s.currentBundles, ref)
	}

	delete(s.stagedBundles, key)
}
