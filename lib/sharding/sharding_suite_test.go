package sharding_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme *runtime.Scheme
)

func TestSharding(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sharding Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")

	ctrl.SetLogger(klog.Background())

	var err error
	scheme, err = setupScheme()
	Expect(err).To(BeNil())
})

func setupScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, err
	}

	return s, nil
}

func randomString() string {
	const length = 10
	return util.RandomString(length)
}
