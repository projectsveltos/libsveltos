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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/cluster-api/util"
)

func TestDeployer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployer Suite")

	ctrl.SetLogger(klog.Background())
}

func randomString() string {
	const length = 10
	return util.RandomString(length)
}
