package roles_test

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	sveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/roles"
)

var _ = Describe("Roles", func() {
	It("GetSecret returns  nil when secret does not exist", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountName := randomString()

		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		secret, err := roles.GetSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountName, sveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(secret).To(BeNil())
	})

	It("GetSecret returns existing secret", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountName := randomString()
		kubeconfig := []byte(randomString())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:        clusterName,
					roles.ServiceAccountNameLabel: serviceaccountName,
				},
			},
			Data: map[string][]byte{
				roles.Key: kubeconfig,
			},
		}

		initObjects := []client.Object{secret}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		currentSecret, err := roles.GetSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountName, sveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(currentSecret).ToNot(BeNil())
		Expect(currentSecret.Namespace).To(Equal(clusterNamespace))
		Expect(currentSecret.Name).To(Equal(secret.Name))

		Expect(currentSecret.Labels).ToNot(BeNil())
		v, ok := currentSecret.Labels[roles.ServiceAccountNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceaccountName))

		v, ok = currentSecret.Labels[roles.ClusterNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(clusterName))

		Expect(currentSecret.Data).ToNot(BeNil())
		var currentKubeconfig []byte
		currentKubeconfig, ok = currentSecret.Data[roles.Key]
		Expect(ok).To(BeTrue())
		Expect(reflect.DeepEqual(currentKubeconfig, kubeconfig)).To(BeTrue())
	})

	It("CreateSecret creates secret and returns it", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountName := randomString()

		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		secret, err := roles.CreateSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountName, sveltosv1alpha1.ClusterTypeSveltos,
			[]byte(randomString()))
		Expect(err).To(BeNil())
		Expect(secret).ToNot(BeNil())
		Expect(secret.Namespace).To(Equal(clusterNamespace))

		Expect(secret.Labels).ToNot(BeNil())
		v, ok := secret.Labels[roles.ServiceAccountNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceaccountName))

		v, ok = secret.Labels[roles.ClusterNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(clusterName))
	})

	It("CreateSecret returns existing secret updating data section", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountName := randomString()
		kubeconfig := []byte(randomString())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:        clusterName,
					roles.ServiceAccountNameLabel: serviceaccountName,
				},
			},
			Data: map[string][]byte{
				roles.Key: []byte(randomString()),
			},
		}

		initObjects := []client.Object{secret}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		currentSecret, err := roles.CreateSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountName, sveltosv1alpha1.ClusterTypeSveltos,
			kubeconfig)
		Expect(err).To(BeNil())
		Expect(currentSecret).ToNot(BeNil())
		Expect(currentSecret.Namespace).To(Equal(clusterNamespace))
		Expect(currentSecret.Name).To(Equal(secret.Name))

		Expect(currentSecret.Labels).ToNot(BeNil())
		v, ok := currentSecret.Labels[roles.ServiceAccountNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceaccountName))

		v, ok = currentSecret.Labels[roles.ClusterNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(clusterName))

		Expect(currentSecret.Data).ToNot(BeNil())
		var currentKubeconfig []byte
		currentKubeconfig, ok = currentSecret.Data[roles.Key]
		Expect(ok).To(BeTrue())
		Expect(reflect.DeepEqual(currentKubeconfig, kubeconfig)).To(BeTrue())
	})

	It("DeleteSecret succeeds when secret does not exist", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountName := randomString()

		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := roles.DeleteSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountName, sveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
	})

	It("DeleteSecret deletes existing secret", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountName := randomString()

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:        clusterName,
					roles.ServiceAccountNameLabel: serviceaccountName,
				},
			},
		}

		initObjects := []client.Object{secret}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		err := roles.DeleteSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountName, sveltosv1alpha1.ClusterTypeSveltos)
		Expect(err).To(BeNil())

		listOptions := []client.ListOption{
			client.InNamespace(clusterNamespace),
			client.MatchingLabels{
				roles.ClusterNameLabel:        clusterName,
				roles.ServiceAccountNameLabel: serviceaccountName,
			},
		}

		secretList := &corev1.SecretList{}
		Expect(c.List(context.TODO(), secretList, listOptions...)).To(Succeed())
		Expect(len(secretList.Items)).To(BeZero())
	})
})
