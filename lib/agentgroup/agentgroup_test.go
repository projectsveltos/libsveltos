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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/agentgroup"
)

const (
	agentNamespace = "projectsveltos"
	component      = "drift-detection"
	clusterNs      = "team-a"
)

var _ = Describe("AgentGroup", func() {
	var (
		ctx  context.Context
		c    client.Client
		pool agentgroup.Pool
	)

	BeforeEach(func() {
		ctx = context.TODO()
		scheme := runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		c = fake.NewClientBuilder().WithScheme(scheme).Build()
		pool = agentgroup.Pool{Component: component, ClusterNamespace: clusterNs}
	})

	member := func(name string) agentgroup.Member {
		return agentgroup.Member{Name: name, ClusterType: libsveltosv1beta1.ClusterTypeSveltos}
	}

	assign := func(client client.Client, p agentgroup.Pool, name string, capacity int) *agentgroup.AssignResult {
		result, err := agentgroup.AssignCluster(ctx, client, client, agentNamespace, p, member(name), capacity)
		Expect(err).To(BeNil())
		return result
	}

	listGroups := func(clusterNamespace string) []agentgroup.Group {
		groups, err := agentgroup.ListGroups(ctx, c, agentNamespace, component, clusterNamespace)
		Expect(err).To(BeNil())
		return groups
	}

	It("creates a group for the first cluster and labels it", func() {
		result := assign(c, pool, "c1", 3)
		Expect(result.Created).To(BeTrue())
		Expect(result.GroupName).To(Equal(pool.GroupName(0)))

		configMap := &corev1.ConfigMap{}
		Expect(c.Get(ctx, types.NamespacedName{Namespace: agentNamespace, Name: result.GroupName}, configMap)).To(Succeed())
		Expect(configMap.Labels).To(Equal(pool.Labels()))

		members, err := agentgroup.ParseMembers(configMap)
		Expect(err).To(BeNil())
		Expect(members).To(Equal([]agentgroup.Member{member("c1")}))
	})

	It("fills a group up to capacity before creating the next one", func() {
		const capacity = 2
		first := assign(c, pool, "c1", capacity)
		second := assign(c, pool, "c2", capacity)
		third := assign(c, pool, "c3", capacity)

		Expect(second.GroupName).To(Equal(first.GroupName))
		Expect(second.Created).To(BeFalse())
		Expect(third.GroupName).To(Equal(pool.GroupName(1)))
		Expect(third.Created).To(BeTrue())
	})

	It("is idempotent for a cluster already in a group", func() {
		first := assign(c, pool, "c1", 2)
		again := assign(c, pool, "c1", 2)

		Expect(again.GroupName).To(Equal(first.GroupName))
		Expect(again.Created).To(BeFalse())
		Expect(listGroups(clusterNs)).To(HaveLen(1))
		Expect(listGroups(clusterNs)[0].Members).To(HaveLen(1))
	})

	It("does not move clusters when capacity is lowered", func() {
		assign(c, pool, "c1", 3)
		assign(c, pool, "c2", 3)
		assign(c, pool, "c3", 3)

		// With capacity 1 the full group stays as it is and only new clusters go elsewhere
		again := assign(c, pool, "c2", 1)
		Expect(again.GroupName).To(Equal(pool.GroupName(0)))
		Expect(listGroups(clusterNs)[0].Members).To(HaveLen(3))

		fresh := assign(c, pool, "c4", 1)
		Expect(fresh.GroupName).To(Equal(pool.GroupName(1)))
	})

	It("keeps the member watchNamespaces up to date", func() {
		assign(c, pool, "c1", 3)

		updated := member("c1")
		updated.WatchNamespaces = []string{"ns1", "ns2"}
		_, err := agentgroup.AssignCluster(ctx, c, c, agentNamespace, pool, updated, 3)
		Expect(err).To(BeNil())

		groups := listGroups(clusterNs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Members).To(Equal([]agentgroup.Member{updated}))
	})

	It("keeps clusters of different namespaces and shards in different groups", func() {
		otherNamespace := agentgroup.Pool{Component: component, ClusterNamespace: "team-b"}
		shardA := agentgroup.Pool{Component: component, ClusterNamespace: clusterNs, Shard: "shard-a"}
		shardB := agentgroup.Pool{Component: component, ClusterNamespace: clusterNs, Shard: "shard-b"}

		names := map[string]struct{}{}
		for _, p := range []agentgroup.Pool{pool, otherNamespace, shardA, shardB} {
			names[assign(c, p, "c1", 10).GroupName] = struct{}{}
		}

		Expect(names).To(HaveLen(4))
	})

	It("reuses the slot of a removed cluster before creating a new group", func() {
		assign(c, pool, "c1", 2)
		assign(c, pool, "c2", 2)
		assign(c, pool, "c3", 2)

		emptied, err := agentgroup.RemoveCluster(ctx, c, c, agentNamespace, component, clusterNs, "c1",
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(emptied).To(BeEmpty())

		result := assign(c, pool, "c4", 2)
		Expect(result.GroupName).To(Equal(pool.GroupName(0)))
		Expect(result.Created).To(BeFalse())
	})

	It("deletes a group when its last member is removed and reports it", func() {
		first := assign(c, pool, "c1", 2)
		assign(c, pool, "c2", 2)

		emptied, err := agentgroup.RemoveCluster(ctx, c, c, agentNamespace, component, clusterNs, "c1",
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(emptied).To(BeEmpty())
		Expect(listGroups(clusterNs)).To(HaveLen(1))

		emptied, err = agentgroup.RemoveCluster(ctx, c, c, agentNamespace, component, clusterNs, "c2",
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(emptied).To(ConsistOf(first.GroupName))
		Expect(listGroups(clusterNs)).To(BeEmpty())
	})

	It("is idempotent when removing a cluster that is in no group", func() {
		emptied, err := agentgroup.RemoveCluster(ctx, c, c, agentNamespace, component, clusterNs, "ghost",
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(emptied).To(BeEmpty())
	})

	It("does not delete a group that gets a new member while its last one is removed", func() {
		first := assign(c, pool, "c1", 2)

		// Right before the group is deleted, another writer adds a member: the delete (carrying
		// the resourceVersion read earlier) must fail and RemoveCluster must look again.
		injected := false
		racing := fake.NewClientBuilder().WithScheme(c.Scheme()).WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, cl client.WithWatch, obj client.Object,
				opts ...client.DeleteOption) error {

				if !injected {
					injected = true
					_, err := agentgroup.AssignCluster(ctx, cl, cl, agentNamespace, pool, member("c2"), 2)
					Expect(err).To(BeNil())
				}
				return cl.Delete(ctx, obj, opts...)
			},
		}).Build()
		// Same content as c, in a client with the interceptor
		configMap := &corev1.ConfigMap{}
		Expect(c.Get(ctx, types.NamespacedName{Namespace: agentNamespace, Name: first.GroupName}, configMap)).To(Succeed())
		configMap.ResourceVersion = ""
		Expect(racing.Create(ctx, configMap)).To(Succeed())

		emptied, err := agentgroup.RemoveCluster(ctx, racing, racing, agentNamespace, component, clusterNs, "c1",
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(emptied).To(BeEmpty())

		groups, err := agentgroup.ListGroups(ctx, racing, agentNamespace, component, clusterNs)
		Expect(err).To(BeNil())
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Members).To(Equal([]agentgroup.Member{member("c2")}))
	})

	It("places every cluster in exactly one group, within capacity, under concurrency", func() {
		const (
			clusters = 40
			capacity = 6
		)

		var wg sync.WaitGroup
		for i := 0; i < clusters; i++ {
			wg.Add(1)
			go func(i int) {
				defer GinkgoRecover()
				defer wg.Done()
				assign(c, pool, fmt.Sprintf("c%d", i), capacity)
			}(i)
		}
		wg.Wait()

		seen := map[string]int{}
		for _, group := range listGroups(clusterNs) {
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

	It("reports an error for a capacity lower than one", func() {
		_, err := agentgroup.AssignCluster(ctx, c, c, agentNamespace, pool, member("c1"), 0)
		Expect(err).ToNot(BeNil())
		Expect(apierrors.IsNotFound(err)).To(BeFalse())
	})
})
