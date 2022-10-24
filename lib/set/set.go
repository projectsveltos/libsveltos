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

package set

import (
	libv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
)

type Set struct {
	data map[libv1alpha1.PolicyRef]bool
}

func (s *Set) init() {
	if s.data == nil {
		s.data = make(map[libv1alpha1.PolicyRef]bool, 0)
	}
}

// Insert adds entry to set
func (s *Set) Insert(entry *libv1alpha1.PolicyRef) {
	s.init()
	s.data[*entry] = true
}

// Erase removes entry from set
func (s *Set) Erase(entry *libv1alpha1.PolicyRef) {
	s.init()
	delete(s.data, *entry)
}

// Has returns true if entry is currently part of set
func (s *Set) Has(entry *libv1alpha1.PolicyRef) bool {
	s.init()
	_, ok := s.data[*entry]
	return ok
}

// Len returns length of set
func (s *Set) Len() int {
	return len(s.data)
}

// Items returns a slice with all elements currently in set
func (s *Set) Items() []libv1alpha1.PolicyRef {
	keys := make([]libv1alpha1.PolicyRef, s.Len())

	i := 0
	for k := range s.data {
		keys[i] = k
		i++
	}

	return keys
}

// Difference returns all elements which are in s but not in b
func (s *Set) Difference(b *Set) []libv1alpha1.PolicyRef {
	results := make([]libv1alpha1.PolicyRef, 0)
	for entry := range s.data {
		if !b.Has(&entry) {
			results = append(results, entry)
		}
	}

	return results
}
