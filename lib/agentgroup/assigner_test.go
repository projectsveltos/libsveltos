/*
Copyright 2026. projectsveltos.io. All rights reserved.

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

package agentgroup_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/agentgroup"
)

var _ = Describe("Assigner", func() {
	var (
		ctx  context.Context
		pool agentgroup.Pool

		scheme *runtime.Scheme

		// api is the "management cluster": the Assigner talks to it through counting clients
		api    client.WithWatch
		lists  atomic.Int32
		writes atomic.Int32
	)

	BeforeEach(func() {
		ctx = context.TODO()
		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		api = fake.NewClientBuilder().WithScheme(scheme).Build()
		pool = agentgroup.Pool{Component: component, ClusterNamespace: clusterNs}
		lists.Store(0)
		writes.Store(0)
	})

	member := func(name string) agentgroup.Member {
		return agentgroup.Member{Name: name, ClusterType: libsveltosv1beta1.ClusterTypeSveltos}
	}

	// newAssigner returns an Assigner over api. Every call to the API server is counted.
	newAssigner := func() *agentgroup.Assigner {
		counting := interceptor.NewClient(api, interceptor.Funcs{
			List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList,
				opts ...client.ListOption) error {

				lists.Add(1)
				return cl.List(ctx, list, opts...)
			},
			Create: func(ctx context.Context, cl client.WithWatch, obj client.Object,
				opts ...client.CreateOption) error {

				writes.Add(1)
				return cl.Create(ctx, obj, opts...)
			},
			Update: func(ctx context.Context, cl client.WithWatch, obj client.Object,
				opts ...client.UpdateOption) error {

				writes.Add(1)
				return cl.Update(ctx, obj, opts...)
			},
			Delete: func(ctx context.Context, cl client.WithWatch, obj client.Object,
				opts ...client.DeleteOption) error {

				writes.Add(1)
				return cl.Delete(ctx, obj, opts...)
			},
		})
		return agentgroup.NewAssigner(counting, counting, agentNamespace)
	}

	groupsOf := func() []agentgroup.Group {
		groups, err := agentgroup.ListGroups(ctx, api, agentNamespace, component, clusterNs)
		Expect(err).To(BeNil())
		return groups
	}

	It("answers a cluster already in a group without calling the API server", func() {
		assigner := newAssigner()

		first, err := assigner.AssignCluster(ctx, pool, member("c1"), 2)
		Expect(err).To(BeNil())
		Expect(first.Created).To(BeTrue())

		// The first call after a change reads the groups again, the following ones do not
		again, err := assigner.AssignCluster(ctx, pool, member("c1"), 2)
		Expect(err).To(BeNil())
		Expect(again.GroupName).To(Equal(first.GroupName))
		Expect(again.Created).To(BeFalse())

		lists.Store(0)
		writes.Store(0)
		for i := 0; i < 10; i++ {
			result, err := assigner.AssignCluster(ctx, pool, member("c1"), 2)
			Expect(err).To(BeNil())
			Expect(result.GroupName).To(Equal(first.GroupName))
		}
		Expect(lists.Load()).To(BeZero())
		Expect(writes.Load()).To(BeZero())
	})

	It("rebuilds what it knows from the ConfigMaps when a new Assigner (a restart) is created", func() {
		before := newAssigner()
		first, err := before.AssignCluster(ctx, pool, member("c1"), 2)
		Expect(err).To(BeNil())
		_, err = before.AssignCluster(ctx, pool, member("c2"), 2)
		Expect(err).To(BeNil())
		_, err = before.AssignCluster(ctx, pool, member("c3"), 2)
		Expect(err).To(BeNil())
		Expect(groupsOf()).To(HaveLen(2))

		after := newAssigner()
		writes.Store(0)

		again, err := after.AssignCluster(ctx, pool, member("c2"), 2)
		Expect(err).To(BeNil())
		Expect(again.GroupName).To(Equal(first.GroupName))
		Expect(again.Created).To(BeFalse())
		Expect(writes.Load()).To(BeZero(), "a cluster already in a group must not be written again")

		// Nothing is lost and no cluster moved: c4 goes to the group with room (the second one)
		fresh, err := after.AssignCluster(ctx, pool, member("c4"), 2)
		Expect(err).To(BeNil())
		Expect(fresh.GroupName).To(Equal(pool.GroupName(1)))
		Expect(groupsOf()).To(HaveLen(2))
	})

	It("places the same cluster in exactly one group when asked concurrently, with clusters leaving", func() {
		const (
			capacity = 2
			workers  = 30
		)

		assigner := newAssigner()
		for i := 0; i < 6; i++ {
			_, err := assigner.AssignCluster(ctx, pool, member(fmt.Sprintf("old%d", i)), capacity)
			Expect(err).To(BeNil())
		}

		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(2)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				_, err := assigner.AssignCluster(ctx, pool, member("target"), capacity)
				Expect(err).To(BeNil())
			}()
			go func(i int) {
				defer GinkgoRecover()
				defer wg.Done()
				if i < 6 {
					_, err := assigner.RemoveCluster(ctx, component, clusterNs, fmt.Sprintf("old%d", i),
						libsveltosv1beta1.ClusterTypeSveltos)
					Expect(err).To(BeNil())
				}
			}(i)
		}
		wg.Wait()

		count := 0
		for _, group := range groupsOf() {
			Expect(len(group.Members)).To(BeNumerically("<=", capacity))
			for _, m := range group.Members {
				if m.Name == "target" {
					count++
				}
			}
		}
		Expect(count).To(Equal(1))
	})

	It("places every cluster in exactly one group, within capacity, under concurrency", func() {
		const (
			clusters = 40
			capacity = 6
		)

		assigner := newAssigner()
		var wg sync.WaitGroup
		for i := 0; i < clusters; i++ {
			wg.Add(1)
			go func(i int) {
				defer GinkgoRecover()
				defer wg.Done()
				_, err := assigner.AssignCluster(ctx, pool, member(fmt.Sprintf("c%d", i)), capacity)
				Expect(err).To(BeNil())
			}(i)
		}
		wg.Wait()

		seen := map[string]int{}
		for _, group := range groupsOf() {
			Expect(len(group.Members)).To(BeNumerically("<=", capacity))
			for _, m := range group.Members {
				seen[m.Name]++
			}
		}
		Expect(seen).To(HaveLen(clusters))
		for name, count := range seen {
			Expect(count).To(Equal(1), fmt.Sprintf("cluster %s is in %d groups", name, count))
		}
	})

	It("recovers when someone else changed a group: the groups are read again", func() {
		assigner := newAssigner()
		_, err := assigner.AssignCluster(ctx, pool, member("c1"), 3)
		Expect(err).To(BeNil())
		// Loaded in memory
		_, err = assigner.AssignCluster(ctx, pool, member("c1"), 3)
		Expect(err).To(BeNil())

		// Another writer adds c2 to the group behind the Assigner's back
		other := agentgroup.NewAssigner(api, api, agentNamespace)
		result, err := other.AssignCluster(ctx, pool, member("c2"), 3)
		Expect(err).To(BeNil())
		Expect(result.GroupName).To(Equal(pool.GroupName(0)))

		// The write of the Assigner carries an old resourceVersion: it conflicts, the groups are read
		// again and c3 is added next to c1 and c2 instead of overwriting c2
		_, err = assigner.AssignCluster(ctx, pool, member("c3"), 3)
		Expect(err).To(BeNil())

		groups := groupsOf()
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Members).To(HaveLen(3))
	})

	It("notices a group deleted by someone else after the resync period", func() {
		assigner := newAssigner()
		assigner.SetResyncPeriod(50 * time.Millisecond)

		first, err := assigner.AssignCluster(ctx, pool, member("c1"), 2)
		Expect(err).To(BeNil())
		_, err = assigner.AssignCluster(ctx, pool, member("c1"), 2)
		Expect(err).To(BeNil())

		configMap := &corev1.ConfigMap{}
		Expect(api.Get(ctx, types.NamespacedName{Namespace: agentNamespace, Name: first.GroupName}, configMap)).To(Succeed())
		Expect(api.Delete(ctx, configMap)).To(Succeed())

		Eventually(func() bool {
			result, err := assigner.AssignCluster(ctx, pool, member("c1"), 2)
			return err == nil && result.Created
		}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())
		Expect(groupsOf()).To(HaveLen(1))
	})

	It("removes a cluster, reports the emptied group and frees the slot", func() {
		assigner := newAssigner()
		first, err := assigner.AssignCluster(ctx, pool, member("c1"), 2)
		Expect(err).To(BeNil())
		_, err = assigner.AssignCluster(ctx, pool, member("c2"), 2)
		Expect(err).To(BeNil())

		emptied, err := assigner.RemoveCluster(ctx, component, clusterNs, "c1", libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(emptied).To(BeEmpty())

		// c3 takes the slot of c1
		result, err := assigner.AssignCluster(ctx, pool, member("c3"), 2)
		Expect(err).To(BeNil())
		Expect(result.GroupName).To(Equal(first.GroupName))

		for _, name := range []string{"c2", "c3"} {
			emptied, err = assigner.RemoveCluster(ctx, component, clusterNs, name, libsveltosv1beta1.ClusterTypeSveltos)
			Expect(err).To(BeNil())
		}
		Expect(emptied).To(ConsistOf(first.GroupName))
		Expect(groupsOf()).To(BeEmpty())
	})

	It("removing a cluster that is in no group does not call the API server once the groups are known", func() {
		assigner := newAssigner()
		_, err := assigner.AssignCluster(ctx, pool, member("c1"), 2)
		Expect(err).To(BeNil())

		// Reads the groups once
		_, err = assigner.RemoveCluster(ctx, component, clusterNs, "ghost", libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())

		lists.Store(0)
		emptied, err := assigner.RemoveCluster(ctx, component, clusterNs, "ghost", libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(emptied).To(BeEmpty())
		Expect(lists.Load()).To(BeZero())
	})

	It("reports an error for a capacity lower than one", func() {
		_, err := newAssigner().AssignCluster(ctx, pool, member("c1"), 0)
		Expect(err).ToNot(BeNil())
	})
})
