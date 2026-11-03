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

package stress_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/agentgroup"
)

// apiCalls counts what is sent to the API server, and the answers that mean "someone else was faster".
type apiCalls struct {
	lists, creates, updates, deletes atomic.Int64
	conflicts, alreadyExists         atomic.Int64
}

func (a *apiCalls) String() string {
	return fmt.Sprintf("list=%d create=%d update=%d delete=%d | conflicts=%d alreadyExists=%d",
		a.lists.Load(), a.creates.Load(), a.updates.Load(), a.deletes.Load(),
		a.conflicts.Load(), a.alreadyExists.Load())
}

// newClient returns a client for the API server. When calls is not nil, it counts what it sends.
func newClient(calls *apiCalls) client.WithWatch {
	c, err := client.NewWithWatch(restConfig, client.Options{Scheme: scheme})
	Expect(err).To(BeNil())
	if calls == nil {
		return c
	}

	return interceptor.NewClient(c, interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			calls.lists.Add(1)
			return cl.List(ctx, list, opts...)
		},
		Create: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			calls.creates.Add(1)
			err := cl.Create(ctx, obj, opts...)
			if apierrors.IsAlreadyExists(err) {
				calls.alreadyExists.Add(1)
			}
			return err
		},
		Update: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			calls.updates.Add(1)
			err := cl.Update(ctx, obj, opts...)
			if apierrors.IsConflict(err) {
				calls.conflicts.Add(1)
			}
			return err
		},
		Delete: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			calls.deletes.Add(1)
			err := cl.Delete(ctx, obj, opts...)
			if apierrors.IsConflict(err) {
				calls.conflicts.Add(1)
			}
			return err
		},
	})
}

type clusterRef struct {
	namespace, name string
}

func (c clusterRef) String() string { return c.namespace + "/" + c.name }

// latencies collects how long each operation took.
type latencies struct {
	mu      sync.Mutex
	samples []time.Duration
}

func (l *latencies) add(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.samples = append(l.samples, d)
}

func (l *latencies) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.samples) == 0 {
		return "no samples"
	}
	sort.Slice(l.samples, func(i, j int) bool { return l.samples[i] < l.samples[j] })
	at := func(p float64) time.Duration { return l.samples[int(float64(len(l.samples)-1)*p)] }
	return fmt.Sprintf("n=%d p50=%v p99=%v max=%v", len(l.samples), at(0.5), at(0.99), l.samples[len(l.samples)-1])
}

var _ = Describe("Assigner against a real API server", Ordered, func() {
	var (
		ctx        = context.TODO()
		clusters   int
		namespaces int
		workers    int
		capacity   int
		component  string
		runNumber  int
	)

	BeforeAll(func() {
		clusters = envInt("AGENTGROUP_STRESS_CLUSTERS", 2000)
		namespaces = envInt("AGENTGROUP_STRESS_NAMESPACES", 20)
		workers = envInt("AGENTGROUP_STRESS_WORKERS", 50)
		capacity = envInt("AGENTGROUP_STRESS_CAPACITY", 5)
		GinkgoWriter.Printf("clusters=%d namespaces=%d workers=%d capacity=%d\n", clusters, namespaces, workers, capacity)
	})

	// Every test uses its own component, so groups of a test do not interfere with the next one
	BeforeEach(func() {
		runNumber++
		component = fmt.Sprintf("stress-%d", runNumber)
	})

	newClusters := func(prefix string, count int) []clusterRef {
		result := make([]clusterRef, 0, count)
		for i := 0; i < count; i++ {
			result = append(result, clusterRef{
				namespace: fmt.Sprintf("team-%d", i%namespaces),
				name:      fmt.Sprintf("%s-%d", prefix, i),
			})
		}
		return result
	}

	member := func(c clusterRef) agentgroup.Member {
		return agentgroup.Member{Name: c.name, ClusterType: libsveltosv1beta1.ClusterTypeSveltos}
	}

	// run executes the jobs with the given number of workers and waits for all of them
	run := func(jobs []func(worker int)) {
		queue := make(chan func(int), len(jobs))
		for _, job := range jobs {
			queue <- job
		}
		close(queue)

		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer GinkgoRecover()
				defer wg.Done()
				for job := range queue {
					job(w)
				}
			}(w)
		}
		wg.Wait()
	}

	assignJobs := func(assigners []*agentgroup.Assigner, refs []clusterRef, copies int, lat *latencies) []func(int) {
		jobs := []func(int){}
		for _, ref := range refs {
			// The same cluster asked several times at once: several ClusterSummaries match it
			for i := 0; i < copies; i++ {
				jobs = append(jobs, func(w int) {
					start := time.Now()
					pool := agentgroup.Pool{Component: component, ClusterNamespace: ref.namespace}
					_, err := assigners[w%len(assigners)].AssignCluster(ctx, pool, member(ref), capacity)
					Expect(err).To(BeNil(), ref.String())
					lat.add(time.Since(start))
				})
			}
		}
		return jobs
	}

	removeJobs := func(assigners []*agentgroup.Assigner, refs []clusterRef, lat *latencies) []func(int) {
		jobs := []func(int){}
		for _, ref := range refs {
			jobs = append(jobs, func(w int) {
				start := time.Now()
				_, err := assigners[w%len(assigners)].RemoveCluster(ctx, component, ref.namespace, ref.name,
					libsveltosv1beta1.ClusterTypeSveltos)
				Expect(err).To(BeNil(), ref.String())
				lat.add(time.Since(start))
			})
		}
		return jobs
	}

	shuffle := func(jobs []func(int)) {
		//nolint:gosec // only shuffles the order of test jobs
		rand.Shuffle(len(jobs), func(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] })
	}

	// groupsOfClusters returns, for each cluster, the groups listing it. It also returns the number
	// of groups, how many are over capacity and how many are empty.
	groupsOfClusters := func() (map[string][]string, int, int, int) {
		reader := newClient(nil)
		perCluster := map[string][]string{}
		total, overCapacity, empty := 0, 0, 0
		for n := 0; n < namespaces; n++ {
			groups, err := agentgroup.ListGroups(ctx, reader, agentNamespace, component, fmt.Sprintf("team-%d", n))
			Expect(err).To(BeNil())
			for _, group := range groups {
				total++
				if len(group.Members) > capacity {
					overCapacity++
				}
				if len(group.Members) == 0 {
					empty++
				}
				for _, m := range group.Members {
					key := group.ClusterNamespace + "/" + m.Name
					perCluster[key] = append(perCluster[key], group.Name)
				}
			}
		}
		return perCluster, total, overCapacity, empty
	}

	// verifyExactly checks that the clusters in expected, and only those, are in exactly one group
	verifyExactly := func(expected []clusterRef) {
		perCluster, total, overCapacity, empty := groupsOfClusters()
		GinkgoWriter.Printf("groups=%d clusters in groups=%d\n", total, len(perCluster))

		Expect(overCapacity).To(BeZero(), "groups over capacity")
		Expect(empty).To(BeZero(), "empty groups left behind")
		Expect(perCluster).To(HaveLen(len(expected)))
		for _, ref := range expected {
			Expect(perCluster[ref.String()]).To(HaveLen(1), "%s is in %v", ref, perCluster[ref.String()])
		}
	}

	report := func(title string, started time.Time, calls *apiCalls, lat *latencies) {
		GinkgoWriter.Printf("%s: %v\n  api: %s\n  latency: %s\n", title, time.Since(started).Round(time.Millisecond), calls, lat)
	}

	It("places every cluster once, asked several times at once, by one process", func() {
		calls := &apiCalls{}
		lat := &latencies{}
		assigners := []*agentgroup.Assigner{agentgroup.NewAssigner(newClient(calls), newClient(calls), agentNamespace)}

		refs := newClusters("c", clusters)
		jobs := assignJobs(assigners, refs, 3, lat)
		shuffle(jobs)

		started := time.Now()
		run(jobs)
		report("one process", started, calls, lat)

		verifyExactly(refs)
	})

	It("places every cluster once with two processes racing on the same pools (handover)", func() {
		calls := &apiCalls{}
		lat := &latencies{}
		assigners := []*agentgroup.Assigner{
			agentgroup.NewAssigner(newClient(calls), newClient(calls), agentNamespace),
			agentgroup.NewAssigner(newClient(calls), newClient(calls), agentNamespace),
		}

		refs := newClusters("c", clusters)
		jobs := assignJobs(assigners, refs, 3, lat)
		shuffle(jobs)

		started := time.Now()
		run(jobs)
		report("two processes, additions only", started, calls, lat)

		verifyExactly(refs)
	})

	It("keeps the groups right under churn: clusters leaving while others are added and asked again", func() {
		calls := &apiCalls{}
		lat := &latencies{}
		assigners := []*agentgroup.Assigner{agentgroup.NewAssigner(newClient(calls), newClient(calls), agentNamespace)}

		refs := newClusters("c", clusters)
		run(assignJobs(assigners, refs, 1, lat))

		leaving, staying := []clusterRef{}, []clusterRef{}
		for i, ref := range refs {
			if i%3 == 0 {
				leaving = append(leaving, ref)
			} else {
				staying = append(staying, ref)
			}
		}
		arriving := newClusters("n", clusters/2)

		jobs := removeJobs(assigners, leaving, lat)
		jobs = append(jobs, assignJobs(assigners, staying, 2, lat)...)
		jobs = append(jobs, assignJobs(assigners, arriving, 2, lat)...)
		shuffle(jobs)

		started := time.Now()
		run(jobs)
		report("one process, churn", started, calls, lat)

		verifyExactly(append(append([]clusterRef{}, staying...), arriving...))
	})

	It("measures what two processes racing with clusters leaving can do (outside what the Assigner guarantees)", func() {
		calls := &apiCalls{}
		lat := &latencies{}
		assigners := []*agentgroup.Assigner{
			agentgroup.NewAssigner(newClient(calls), newClient(calls), agentNamespace),
			agentgroup.NewAssigner(newClient(calls), newClient(calls), agentNamespace),
		}

		refs := newClusters("c", clusters)
		run(assignJobs(assigners[:1], refs, 1, lat))

		leaving, staying := []clusterRef{}, []clusterRef{}
		for i, ref := range refs {
			if i%3 == 0 {
				leaving = append(leaving, ref)
			} else {
				staying = append(staying, ref)
			}
		}
		arriving := newClusters("n", clusters/2)

		jobs := removeJobs(assigners, leaving, lat)
		jobs = append(jobs, assignJobs(assigners, staying, 2, lat)...)
		jobs = append(jobs, assignJobs(assigners, arriving, 2, lat)...)
		shuffle(jobs)

		started := time.Now()
		run(jobs)
		report("two processes, churn", started, calls, lat)

		perCluster, total, overCapacity, _ := groupsOfClusters()
		duplicates, missing := 0, 0
		for _, ref := range append(append([]clusterRef{}, staying...), arriving...) {
			switch len(perCluster[ref.String()]) {
			case 0:
				missing++
			case 1:
			default:
				duplicates++
			}
		}
		GinkgoWriter.Printf("groups=%d overCapacity=%d clusters in more than one group=%d missing=%d\n",
			total, overCapacity, duplicates, missing)

		// A cluster must never be lost. Being in two groups is possible only with two processes
		// deciding at the same time, which the controller does not do: it is reported, not asserted.
		Expect(missing).To(BeZero())
		Expect(overCapacity).To(BeZero())
	})
})
