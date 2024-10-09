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

package deployer_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	sveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
)

var messages chan string

func writeToChannelHandler(ctx context.Context, c client.Client,
	namespace, name, applicant, featureID string, clusterType sveltosv1beta1.ClusterType,
	o deployer.Options, logger logr.Logger) error {

	By("writeToChannelHandler: writing to channel")
	messages <- "done deploying"
	return nil
}

func metricHandler(elapsed time.Duration,
	clusterNamespace, clusterName, featureID string,
	clusterType sveltosv1beta1.ClusterType,
	logger logr.Logger) {

	By("metricHandler: storing metrics")
}

func doNothingHandler(ctx context.Context, c client.Client,
	namespace, name, applicant, featureID string, clusterType sveltosv1beta1.ClusterType,
	o deployer.Options, logger logr.Logger) error {

	return nil
}

var _ = Describe("Worker", func() {
	It("getKey and all get FromKey return correct values", func() {
		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := true
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)

		outNs, outName, err := deployer.GetClusterFromKey(key)
		Expect(err).To(BeNil())
		outApplicant, outFeatureID, err := deployer.GetApplicatantAndFeatureFromKey(key)
		Expect(err).To(BeNil())
		outCleanup, err := deployer.GetIsCleanupFromKey(key)
		Expect(err).To(BeNil())
		Expect(outNs).To(Equal(ns))
		Expect(outName).To(Equal(name))
		Expect(outApplicant).To(Equal(applicant))
		Expect(outFeatureID).To(Equal(featureID))
		Expect(outCleanup).To(Equal(cleanup))
	})

	It("getKey and get FromKey return correct values (applicant is empty)", func() {
		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := ""
		featureID := randomString()
		cleanup := false
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, false)

		outNs, outName, err := deployer.GetClusterFromKey(key)
		Expect(err).To(BeNil())
		outApplicant, outFeatureID, err := deployer.GetApplicatantAndFeatureFromKey(key)
		Expect(err).To(BeNil())
		outCleanup, err := deployer.GetIsCleanupFromKey(key)
		Expect(err).To(BeNil())
		Expect(outNs).To(Equal(ns))
		Expect(outName).To(Equal(name))
		Expect(outApplicant).To(Equal(applicant))
		Expect(outFeatureID).To(Equal(featureID))
		Expect(outCleanup).To(Equal(cleanup))
	})

	It("removeFromSlice should remove element from slice", func() {
		tmp := []string{"eng", "sale", "hr"}
		tmp = deployer.RemoveFromSlice(tmp, 1)
		Expect(len(tmp)).To(Equal(2))
		Expect(tmp[0]).To(Equal("eng"))
		Expect(tmp[1]).To(Equal("hr"))

		tmp = deployer.RemoveFromSlice(tmp, 1)
		Expect(len(tmp)).To(Equal(1))

		tmp = deployer.RemoveFromSlice(tmp, 0)
		Expect(len(tmp)).To(Equal(0))
	})

	It("storeResult saves results and removes key from inProgress", func() {
		c := fake.NewClientBuilder().WithObjects(nil...).Build()
		d := deployer.GetClient(context.TODO(),
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))), c, 10)
		defer d.ClearInternalStruct()

		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := false
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)
		d.SetInProgress([]string{key})
		Expect(len(d.GetInProgress())).To(Equal(1))

		deployer.StoreResult(d, key, nil, deployer.Options{}, doNothingHandler, metricHandler,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(len(d.GetInProgress())).To(Equal(0))
	})

	It("storeResult saves results and removes key from dirty and adds to jobQueue", func() {
		c := fake.NewClientBuilder().WithObjects(nil...).Build()
		d := deployer.GetClient(context.TODO(), textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))), c, 10)
		defer d.ClearInternalStruct()

		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := false
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)
		d.SetInProgress([]string{key})
		Expect(len(d.GetInProgress())).To(Equal(1))

		d.SetDirty([]string{key})
		Expect(len(d.GetDirty())).To(Equal(1))

		deployer.StoreResult(d, key, nil, deployer.Options{}, doNothingHandler, metricHandler,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(len(d.GetInProgress())).To(Equal(0))
		Expect(len(d.GetDirty())).To(Equal(0))
		Expect(len(d.GetJobQueue())).To(Equal(1))
	})

	It("getRequestStatus returns result when available", func() {
		c := fake.NewClientBuilder().WithObjects(nil...).Build()
		d := deployer.GetClient(context.TODO(),
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))), c, 10)
		defer d.ClearInternalStruct()

		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := true
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeSveltos, cleanup)

		r := map[string]error{key: nil}
		d.SetResults(r)
		Expect(len(d.GetResults())).To(Equal(1))

		resp, err := deployer.GetRequestStatus(d, ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeSveltos, cleanup)
		Expect(err).To(BeNil())
		Expect(resp).ToNot(BeNil())
		Expect(deployer.IsResponseDeployed(resp)).To(BeTrue())
	})

	It("getRequestStatus returns result when available and reports error", func() {
		c := fake.NewClientBuilder().WithObjects(nil...).Build()
		d := deployer.GetClient(context.TODO(),
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))), c, 10)
		defer d.ClearInternalStruct()

		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := true
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)

		resultErr := errors.New("failed to deploy")
		r := map[string]error{key: resultErr}
		d.SetResults(r)
		Expect(len(d.GetResults())).To(Equal(1))

		resp, err := deployer.GetRequestStatus(d, ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)
		Expect(err).To(BeNil())
		Expect(resp).ToNot(BeNil())
		Expect(deployer.IsResponseFailed(resp)).To(BeTrue())
		Expect(deployer.GetResponseError(resp)).To(Equal(resultErr))
	})

	It("getRequestStatus returns nil response when request is still queued (currently in progress)", func() {
		c := fake.NewClientBuilder().WithObjects(nil...).Build()
		d := deployer.GetClient(context.TODO(),
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))), c, 10)
		defer d.ClearInternalStruct()

		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := false
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeSveltos, cleanup)

		d.SetInProgress([]string{key})
		Expect(len(d.GetInProgress())).To(Equal(1))

		resp, err := deployer.GetRequestStatus(d, ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeSveltos, cleanup)
		Expect(err).To(BeNil())
		Expect(resp).To(BeNil())
	})

	It("getRequestStatus returns nil response when request is still queued (currently queued)", func() {
		c := fake.NewClientBuilder().WithObjects(nil...).Build()
		d := deployer.GetClient(context.TODO(),
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))), c, 10)
		defer d.ClearInternalStruct()

		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := false
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)

		d.SetJobQueue(key, nil, nil)
		Expect(len(d.GetJobQueue())).To(Equal(1))

		resp, err := deployer.GetRequestStatus(d, ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)
		Expect(err).To(BeNil())
		Expect(resp).To(BeNil())
	})

	It("processRequests process request and stores results", func() {
		c := fake.NewClientBuilder().WithObjects(nil...).Build()
		ctx, cancel := context.WithCancel(context.TODO())
		d := deployer.GetClient(ctx,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))), c, 10)
		defer d.ClearInternalStruct()

		ns := namespacePrefix + randomString()
		name := namespacePrefix + randomString()
		applicant := randomString()
		featureID := randomString()
		cleanup := true
		key := deployer.GetKey(ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)
		d.SetJobQueue(key, writeToChannelHandler, metricHandler)
		Expect(len(d.GetJobQueue())).To(Equal(1))
		messages = make(chan string)

		go deployer.ProcessRequests(ctx, d, 1,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		gotResult := false
		go func() {
			// wait for processRequest to process the request
			// processRequest processes queued request every second
			<-messages
			By("read from channel. Request is processed")
			gotResult = true
			cancel()
		}()
		// wait for result to be available
		Eventually(func() bool {
			return gotResult
		}, 20*time.Second, time.Second).Should(BeTrue())

		resp, err := deployer.GetRequestStatus(d, ns, name, applicant, featureID, sveltosv1beta1.ClusterTypeCapi, cleanup)
		Expect(err).To(BeNil())
		Expect(deployer.IsResponseDeployed(resp)).To(BeTrue())
	})
})
