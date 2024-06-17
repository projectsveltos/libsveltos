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

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/crd"
	"github.com/projectsveltos/libsveltos/lib/roles"
	"github.com/projectsveltos/libsveltos/lib/utils"
)

var _ = Describe("Roles", func() {
	It("GetKubeconfig returns nil when secret does not exist", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceAccountNamespace := randomString()
		serviceaccountName := randomString()

		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		secret, err := roles.GetKubeconfig(context.TODO(), c,
			clusterNamespace, clusterName, serviceAccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(secret).To(BeNil())
	})

	It("GetKubeconfig returns kubeconfig", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceAccountNamespace := randomString()
		serviceaccountName := randomString()
		kubeconfig := []byte(randomString())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:             clusterName,
					roles.ServiceAccountNameLabel:      serviceaccountName,
					roles.ServiceAccountNamespaceLabel: serviceAccountNamespace,
				},
			},
			Data: map[string][]byte{
				roles.Key: kubeconfig,
			},
		}

		initObjects := []client.Object{secret}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		currentKubeconfig, err := roles.GetKubeconfig(context.TODO(), c,
			clusterNamespace, clusterName, serviceAccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(currentKubeconfig).ToNot(BeNil())
		Expect(reflect.DeepEqual(currentKubeconfig, kubeconfig)).To(BeTrue())
	})

	It("GetSecret returns nil when secret does not exist", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceAccountNamespace := randomString()
		serviceaccountName := randomString()

		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		secret, err := roles.GetSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceAccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(secret).To(BeNil())
	})

	It("GetSecret returns existing secret", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceAccountNamespace := randomString()
		serviceaccountName := randomString()
		kubeconfig := []byte(randomString())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:             clusterName,
					roles.ServiceAccountNameLabel:      serviceaccountName,
					roles.ServiceAccountNamespaceLabel: serviceAccountNamespace,
				},
			},
			Data: map[string][]byte{
				roles.Key: kubeconfig,
			},
		}

		initObjects := []client.Object{secret}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		currentSecret, err := roles.GetSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceAccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(currentSecret).ToNot(BeNil())
		Expect(currentSecret.Namespace).To(Equal(clusterNamespace))
		Expect(currentSecret.Name).To(Equal(secret.Name))

		Expect(currentSecret.Labels).ToNot(BeNil())
		v, ok := currentSecret.Labels[roles.ServiceAccountNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceaccountName))

		v, ok = currentSecret.Labels[roles.ServiceAccountNamespaceLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceAccountNamespace))

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
		serviceAccountNamespace := randomString()
		serviceaccountName := randomString()

		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		roleRequestCRD, err := utils.GetUnstructured(crd.GetRoleRequestCRDYAML())
		Expect(err).To(BeNil())
		Expect(c.Create(context.TODO(), roleRequestCRD)).To(Succeed())

		roleRequest := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}

		secret, err := roles.CreateSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceAccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos, []byte(randomString()), roleRequest)
		Expect(err).To(BeNil())
		Expect(secret).ToNot(BeNil())
		Expect(secret.Namespace).To(Equal(clusterNamespace))

		Expect(secret.Labels).ToNot(BeNil())
		v, ok := secret.Labels[roles.ServiceAccountNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceaccountName))

		v, ok = secret.Labels[roles.ServiceAccountNamespaceLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceAccountNamespace))

		v, ok = secret.Labels[roles.ClusterNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(clusterName))

		Expect(secret.OwnerReferences).ToNot(BeNil())
		Expect(len(secret.OwnerReferences)).To(Equal(1))
	})

	It("CreateSecret returns existing secret updating data section and ownerreference", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceAccountNamespace := randomString()
		serviceaccountName := randomString()
		kubeconfig := []byte(randomString())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:             clusterName,
					roles.ServiceAccountNameLabel:      serviceaccountName,
					roles.ServiceAccountNamespaceLabel: serviceAccountNamespace,
				},
			},
			Data: map[string][]byte{
				roles.Key: []byte(randomString()),
			},
		}

		initObjects := []client.Object{secret}

		roleRequest := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		currentSecret, err := roles.CreateSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceAccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos, kubeconfig, roleRequest)
		Expect(err).To(BeNil())
		Expect(currentSecret).ToNot(BeNil())
		Expect(currentSecret.Namespace).To(Equal(clusterNamespace))
		Expect(currentSecret.Name).To(Equal(secret.Name))

		Expect(currentSecret.Labels).ToNot(BeNil())
		v, ok := currentSecret.Labels[roles.ServiceAccountNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceaccountName))

		v, ok = currentSecret.Labels[roles.ServiceAccountNamespaceLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(serviceAccountNamespace))

		v, ok = currentSecret.Labels[roles.ClusterNameLabel]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(clusterName))

		Expect(currentSecret.Data).ToNot(BeNil())
		var currentKubeconfig []byte
		currentKubeconfig, ok = currentSecret.Data[roles.Key]
		Expect(ok).To(BeTrue())
		Expect(reflect.DeepEqual(currentKubeconfig, kubeconfig)).To(BeTrue())

		Expect(currentSecret.OwnerReferences).ToNot(BeNil())
		Expect(len(currentSecret.OwnerReferences)).To(Equal(1))
	})

	It("DeleteSecret succeeds when secret does not exist", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountNamespace := randomString()
		serviceaccountName := randomString()

		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		roleRequest := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, roleRequest)).To(Succeed())

		err := roles.DeleteSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos, roleRequest)
		Expect(err).To(BeNil())
	})

	It("DeleteSecret deletes existing secret", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountNamespace := randomString()
		serviceaccountName := randomString()

		roleRequest := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, roleRequest)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:             clusterName,
					roles.ServiceAccountNameLabel:      serviceaccountName,
					roles.ServiceAccountNamespaceLabel: serviceaccountNamespace,
				},
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: roleRequest.APIVersion, Kind: libsveltosv1beta1.RoleRequestKind, Name: roleRequest.Name},
				},
			},
		}

		initObjects := []client.Object{secret}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		err := roles.DeleteSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos, roleRequest)
		Expect(err).To(BeNil())

		listOptions := []client.ListOption{
			client.InNamespace(clusterNamespace),
			client.MatchingLabels{
				roles.ClusterNameLabel:             clusterName,
				roles.ServiceAccountNameLabel:      serviceaccountName,
				roles.ServiceAccountNamespaceLabel: serviceaccountNamespace,
			},
		}

		secretList := &corev1.SecretList{}
		Expect(c.List(context.TODO(), secretList, listOptions...)).To(Succeed())
		Expect(len(secretList.Items)).To(BeZero())
	})

	It("DeleteSecret does not delete existing secret with multiple owners", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		serviceaccountNamespace := randomString()
		serviceaccountName := randomString()

		roleRequest1 := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, roleRequest1)).To(Succeed())

		roleRequest2 := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, roleRequest2)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels: map[string]string{
					roles.ClusterNameLabel:             clusterName,
					roles.ServiceAccountNameLabel:      serviceaccountName,
					roles.ServiceAccountNamespaceLabel: serviceaccountNamespace,
				},
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: roleRequest1.APIVersion, Kind: libsveltosv1beta1.RoleRequestKind, Name: roleRequest1.Name},
					{APIVersion: roleRequest2.APIVersion, Kind: libsveltosv1beta1.RoleRequestKind, Name: roleRequest2.Name},
				},
			},
		}

		initObjects := []client.Object{secret}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		err := roles.DeleteSecret(context.TODO(), c,
			clusterNamespace, clusterName, serviceaccountNamespace, serviceaccountName,
			libsveltosv1beta1.ClusterTypeSveltos, roleRequest1)
		Expect(err).To(BeNil())

		listOptions := []client.ListOption{
			client.InNamespace(clusterNamespace),
			client.MatchingLabels{
				roles.ClusterNameLabel:             clusterName,
				roles.ServiceAccountNameLabel:      serviceaccountName,
				roles.ServiceAccountNamespaceLabel: serviceaccountNamespace,
			},
		}

		secretList := &corev1.SecretList{}
		Expect(c.List(context.TODO(), secretList, listOptions...)).To(Succeed())
		Expect(len(secretList.Items)).To(Equal(1))
		Expect(secretList.Items[0].OwnerReferences).ToNot(BeNil())
		Expect(len(secretList.Items[0].OwnerReferences)).To(Equal(1))
		Expect(secretList.Items[0].OwnerReferences[0].Name).To(Equal(roleRequest2.Name))
	})

	It("ListSecretForOwner returns all secret for which owner is one of the OnwerReferences", func() {
		roleRequest1 := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, roleRequest1)).To(Succeed())

		secret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Labels: map[string]string{
					libsveltosv1beta1.RoleRequestLabel: "ok",
				},
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: roleRequest1.APIVersion, Kind: libsveltosv1beta1.RoleRequestKind, Name: roleRequest1.Name},
				},
			},
		}

		secret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: roleRequest1.APIVersion, Kind: libsveltosv1beta1.RoleRequestKind, Name: roleRequest1.Name},
				},
			},
		}

		secret3 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Labels: map[string]string{
					libsveltosv1beta1.RoleRequestLabel: "ok",
				},
			},
		}

		initObjects := []client.Object{secret1, secret2, secret3}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
		list, err := roles.ListSecretForOwner(context.TODO(), c, roleRequest1)
		Expect(err).To(BeNil())
		Expect(len(list)).To(Equal(1))
		Expect(list[0].Name).To(Equal(secret1.Name))
		Expect(list[0].Namespace).To(Equal(secret1.Namespace))
	})

	It("ListSecrets returns all secret created for any RoleRequest", func() {
		initObjects := []client.Object{}

		expectedSecret := 5
		for i := 0; i < expectedSecret; i++ {
			roleRequestSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: randomString(),
					Name:      randomString(),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: libsveltosv1beta1.GroupVersion.String(),
							Kind:       libsveltosv1beta1.RoleRequestKind,
							Name:       randomString()},
					},
					Labels: map[string]string{
						libsveltosv1beta1.RoleRequestLabel: "ok",
					},
				},
			}

			normalSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: randomString(),
					Name:      randomString(),
				},
			}

			initObjects = append(initObjects, roleRequestSecret, normalSecret)
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
		list, err := roles.ListSecrets(context.TODO(), c)
		Expect(err).To(BeNil())
		Expect(len(list)).To(Equal(expectedSecret))
	})
})
