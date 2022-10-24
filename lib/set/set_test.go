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

package set_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	libv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/set"
)

func getEntry() *libv1alpha1.PolicyRef {
	return &libv1alpha1.PolicyRef{
		Kind:      randomString(),
		Namespace: randomString(),
		Name:      randomString(),
	}
}

var _ = Describe("Set", func() {
	It("insert adds entry", func() {
		s := &set.Set{}
		entry := getEntry()
		s.Insert(entry)
		Expect(len(s.Items())).To(Equal(1))
	})

	It("erase removes entry", func() {
		s := &set.Set{}
		entry := getEntry()
		s.Insert(entry)
		Expect(len(s.Items())).To(Equal(1))
		s.Erase(entry)
		Expect(len(s.Items())).To(Equal(0))
	})

	It("len returns number of entries in set", func() {
		s := &set.Set{}
		for i := 0; i < 10; i++ {
			entry := getEntry()
			s.Insert(entry)
			Expect(len(s.Items())).To(Equal(i + 1))
		}
	})

	It("has returns true when entry is in set", func() {
		s := &set.Set{}
		numEntries := 10
		for i := 0; i < numEntries; i++ {
			entry := getEntry()
			s.Insert(entry)
			Expect(len(s.Items())).To(Equal(i + 1))
		}
		entry := getEntry()
		Expect(s.Has(entry)).To(BeFalse())
		s.Insert(entry)
		Expect(len(s.Items())).To(Equal(numEntries + 1))
		Expect(s.Has(entry)).To(BeTrue())
	})

	It("items returns all entries in set", func() {
		s := &set.Set{}
		entry0 := getEntry()
		s.Insert(entry0)
		entry1 := getEntry()
		s.Insert(entry1)
		entries := s.Items()
		Expect(entries).To(ContainElement(*entry0))
		Expect(entries).To(ContainElement(*entry1))
	})
})
