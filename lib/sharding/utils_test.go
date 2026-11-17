package sharding_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/sharding"
)

var _ = Describe("Sharding", func() {
	It("RegisterClusterShard returns correct information with respect to cluster sharding", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		cluster := &corev1.ObjectReference{
			Name:       randomString(),
			Namespace:  randomString(),
			Kind:       libsveltosv1beta1.SveltosClusterKind,
			APIVersion: libsveltosv1beta1.GroupVersion.String(),
		}

		shard := randomString()
		sveltosNamespace := randomString()
		// First time, add entry and return false since cluster:shard was never changed
		shardChanged, err := sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			sveltosNamespace, "helm", shard, cluster.Namespace, cluster.Name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		// return false since cluster:shard was never changed
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			sveltosNamespace, "helm", shard, cluster.Namespace, cluster.Name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		// return true since cluster:shard has changed
		newShard := randomString()
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			sveltosNamespace, "helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeTrue())

		// return false since cluster:shard has not changed (and previous step updated configMap)
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			sveltosNamespace, "helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		// register capi cluster with same namespace/name of sveltoscluster used so far
		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			sveltosNamespace, "helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1beta1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			sveltosNamespace, "helm", newShard, cluster.Namespace, cluster.Name, libsveltosv1beta1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeFalse())

		shardChanged, err = sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			sveltosNamespace, "helm", randomString(), cluster.Namespace, cluster.Name, libsveltosv1beta1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(shardChanged).To(BeTrue())
	})

	It("RegisterClusterShard propagates non-NotFound errors when reading the tracking ConfigMap", func() {
		// A NotFound error means the ConfigMap does not exist yet and must be created.
		// Any other error (for instance a transient API server error or a Forbidden) must
		// be returned to the caller instead of being swallowed, otherwise sharding changes
		// would be silently lost and an empty ConfigMap acted upon.
		forbiddenErr := apierrors.NewForbidden(corev1.Resource("configmaps"), "clustersharding",
			errors.New("denied by test"))

		c := interceptor.NewClient(fake.NewClientBuilder().WithScheme(scheme).Build(),
			interceptor.Funcs{
				Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey,
					_ client.Object, _ ...client.GetOption) error {
					return forbiddenErr
				},
			})

		shardChanged, err := sharding.RegisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			randomString(), "helm", randomString(), randomString(), randomString(), libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsForbidden(err)).To(BeTrue())
		Expect(shardChanged).To(BeFalse())

		err = sharding.UnregisterClusterShard(context.TODO(), c, libsveltosv1beta1.ComponentAddonManager,
			randomString(), "helm", randomString(), randomString(), libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsForbidden(err)).To(BeTrue())
	})

	It("IsShardAMatch returns false when shard is not a match", func() {
		cluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   randomString(),
				Name:        randomString(),
				Annotations: map[string]string{},
			},
		}

		shard := randomString()

		Expect(sharding.IsShardAMatch(shard, cluster)).To(BeFalse())

		cluster.Annotations[libsveltosv1beta1.ShardAnnotation] = randomString()
		Expect(sharding.IsShardAMatch(shard, cluster)).To(BeFalse())
	})

	It("IsShardAMatch returns true when shard is a match", func() {
		shard := randomString()

		cluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Annotations: map[string]string{
					libsveltosv1beta1.ShardAnnotation: shard,
				},
			},
		}

		Expect(sharding.IsShardAMatch(shard, cluster)).To(BeTrue())
	})
})
