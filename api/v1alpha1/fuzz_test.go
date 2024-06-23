/*
Copyright 2024. projectsveltos.io. All rights reserved.

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

package v1alpha1_test

import (
	"fmt"
	"testing"

	fuzz "github.com/google/gofuzz"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

func TestFuzzyConversion(t *testing.T) {
	t.Run("for ClusterHealthCheck", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &libsveltosv1beta1.ClusterHealthCheck{},
		Spoke:       &libsveltosv1alpha1.ClusterHealthCheck{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for RoleRequest", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &libsveltosv1beta1.RoleRequest{},
		Spoke:       &libsveltosv1alpha1.RoleRequest{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for ClusterSet", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &libsveltosv1beta1.ClusterSet{},
		Spoke:       &libsveltosv1alpha1.ClusterSet{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for Set", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &libsveltosv1beta1.Set{},
		Spoke:       &libsveltosv1alpha1.Set{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))
}

func fuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		clusterHealthCheckFuzzer,
		roleRequestClusterSelectorFuzzer,
		clusterSetClusterSelectorFuzzer,
		setClusterSelectorFuzzer,
		v1beta1ClusterHealthCheckClusterSelectorFuzzer,
		v1beta1RoleRequestClusterSelectorFuzzer,
		v1beta1ClusterSetClusterSelectorFuzzer,
		v1beta1SetClusterSelectorFuzzer,
	}
}

func clusterHealthCheckFuzzer(in *libsveltosv1alpha1.ClusterHealthCheck, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1alpha1.Selector(
		fmt.Sprintf("%s=%s",
			randomString(), randomString(),
		))

	in.Spec.LivenessChecks = []libsveltosv1alpha1.LivenessCheck{
		{
			Type: libsveltosv1alpha1.LivenessTypeHealthCheck,
			LivenessSourceRef: &corev1.ObjectReference{
				Namespace:  randomString(),
				Name:       randomString(),
				Kind:       libsveltosv1alpha1.HealthCheckKind,
				APIVersion: libsveltosv1alpha1.GroupVersion.String(),
			},
		},
	}
}

func roleRequestClusterSelectorFuzzer(in *libsveltosv1alpha1.RoleRequest, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1alpha1.Selector(
		fmt.Sprintf("%s=%s",
			randomString(), randomString(),
		))
}

func clusterSetClusterSelectorFuzzer(in *libsveltosv1alpha1.ClusterSet, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1alpha1.Selector(
		fmt.Sprintf("%s=%s",
			randomString(), randomString(),
		))
}

func setClusterSelectorFuzzer(in *libsveltosv1alpha1.Set, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1alpha1.Selector(
		fmt.Sprintf("%s=%s",
			randomString(), randomString(),
		))
}

func v1beta1ClusterHealthCheckClusterSelectorFuzzer(in *libsveltosv1beta1.ClusterHealthCheck, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1beta1.Selector{
		LabelSelector: metav1.LabelSelector{
			MatchExpressions: nil,
			MatchLabels: map[string]string{
				randomString(): randomString(),
			},
		},
	}
}

func v1beta1RoleRequestClusterSelectorFuzzer(in *libsveltosv1beta1.RoleRequest, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1beta1.Selector{
		LabelSelector: metav1.LabelSelector{
			MatchExpressions: nil,
			MatchLabels: map[string]string{
				randomString(): randomString(),
			},
		},
	}
}

func v1beta1ClusterSetClusterSelectorFuzzer(in *libsveltosv1beta1.ClusterSet, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1beta1.Selector{
		LabelSelector: metav1.LabelSelector{
			MatchExpressions: nil,
			MatchLabels: map[string]string{
				randomString(): randomString(),
			},
		},
	}
}

func v1beta1SetClusterSelectorFuzzer(in *libsveltosv1beta1.Set, _ fuzz.Continue) {
	in.Spec.ClusterSelector = libsveltosv1beta1.Selector{
		LabelSelector: metav1.LabelSelector{
			MatchExpressions: nil,
			MatchLabels: map[string]string{
				randomString(): randomString(),
			},
		},
	}
}
