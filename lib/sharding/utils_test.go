package sharding_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/sharding"
)

var _ = Describe("Sharding", func() {
	It("RegisterClusterShard returns correct information with respect to cluster sharding", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		cluster := &corev1.ObjectReference{
			Name:       randomString(),
			Namespace:  randomString(),
			Kind:       libsveltosv1alpha1.SveltosClusterKind,
			APIVersion: libsveltosv1alpha1.GroupVersion.String(),
		}

		shard := randomString()
		// First time, add entry and return false since cluster:shard was never changed
		shardChanged, err := sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1alpha1.ComponentAddonManager,
			"helm", shard, cluster.Namespace, cluster.Name, libsveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		// return false since cluster:shard was never changed
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1alpha1.ComponentAddonManager,
			"helm", shard, cluster.Namespace, cluster.Name, libsveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		// return true since cluster:shard has changed
		newShard := randomString()
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1alpha1.ComponentAddonManager,
			"helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeTrue())

		// return false since cluster:shard has not changed (and previous step updated configMap)
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1alpha1.ComponentAddonManager,
			"helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		// register capi cluster with same namespace/name of sveltoscluster used so far
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1alpha1.ComponentAddonManager,
			"helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1alpha1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1alpha1.ComponentAddonManager,
			"helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1alpha1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1alpha1.ComponentAddonManager,
			"helm", randomString(), cluster.Namespace, cluster.Name, libsveltosv1alpha1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeTrue())
	})

	It("IsShardAMatch returns false when shard is not a match", func() {
		cluster := &libsveltosv1alpha1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   randomString(),
				Name:        randomString(),
				Annotations: map[string]string{},
			},
		}

		shard := randomString()

		Expect(sharding.IsShardAMatch(shard, cluster)).To(BeFalse())

		cluster.Annotations[sharding.ShardAnnotation] = randomString()
		Expect(sharding.IsShardAMatch(shard, cluster)).To(BeFalse())
	})

	It("IsShardAMatch returns true when shard is a match", func() {
		shard := randomString()

		cluster := &libsveltosv1alpha1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Annotations: map[string]string{
					sharding.ShardAnnotation: shard,
				},
			},
		}

		Expect(sharding.IsShardAMatch(shard, cluster)).To(BeTrue())
	})
})
