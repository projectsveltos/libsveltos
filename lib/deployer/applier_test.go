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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2/textlogger"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
)

const (
	multusData = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: network-attachment-definitions.k8s.cni.cncf.io
spec:
  group: k8s.cni.cncf.io
  scope: Namespaced
  names:
    plural: network-attachment-definitions
    singular: network-attachment-definition
    kind: NetworkAttachmentDefinition
    shortNames:
    - net-attach-def
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        description: 'NetworkAttachmentDefinition is a CRD schema specified by the Network Plumbing Working Group
			to express the intent for attaching pods to one or more logical or physical networks.
			More information available at: https://github.com/k8snetworkplumbingwg/multi-net-spec'
        type: object
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this represen tation of an object. Servers
				should convert recognized schemas to the latest internal value, and may reject unrecognized values.
				More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this object represents. Servers
			may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase.
			More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: 'NetworkAttachmentDefinition spec defines the desired state of a network attachment'
            type: object
            properties:
              config:
                description: 'NetworkAttachmentDefinition config is a JSON-formatted CNI configuration'
                type: string
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: multus
  namespace: kube-system
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: multus
rules:
- apiGroups: ["k8s.cni.cncf.io"]
  resources:
  - '*'
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - pods
  - pods/status
  verbs:
  - get
  - update
- apiGroups:
  - ""
  - events.k8s.io
  resources:
  - events
  verbs:
  - create
  - patch
  - update
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: multus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: multus
subjects:
- kind: ServiceAccount
  name: multus
  namespace: kube-system
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-multus-ds
  namespace: kube-system
  labels:
    tier: node
    app: multus
    name: multus
spec:
  selector:
    matchLabels:
      name: multus
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        tier: node
        app: multus
        name: multus
    spec:
      hostNetwork: true
      hostPID: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      - operator: Exists
        effect: NoExecute
      serviceAccountName: multus
      containers:
      - name: kube-multus
        image: ghcr.io/k8snetworkplumbingwg/multus-cni:v4.0.2-thick
        command: ["/usr/src/multus-cni/bin/multus-daemon"]
        resources:
          requests:
            cpu: "100m"
            memory: "200Mi"
          limits:
            cpu: "100m"
            memory: "200Mi"
        securityContext:
          privileged: true
        volumeMounts:
        - name: cni
          mountPath: /host/etc/cni/net.d
        - name: host-run
          mountPath: /host/run
        - name: host-var-lib-cni-multus
          mountPath: /var/lib/cni/multus
        - name: host-var-lib-kubelet
          mountPath: /var/lib/kubelet
        - name: host-run-k8s-cni-cncf-io
          mountPath: /run/k8s.cni.cncf.io
        - name: host-run-netns
          mountPath: /run/netns
          mountPropagation: HostToContainer
        - name: multus-daemon-config
          mountPath: /etc/cni/net.d/multus.d
          readOnly: true
        - name: hostroot
          mountPath: /hostroot
          mountPropagation: HostToContainer
      initContainers:
      - name: install-multus-binary
        image: ghcr.io/k8snetworkplumbingwg/multus-cni:v4.0.2-thick
        command:
        - "cp"
        - "/usr/src/multus-cni/bin/multus-shim"
        - "/host/opt/cni/bin/multus-shim"
        resources:
          requests:
            cpu: "10m"
            memory: "15Mi"
        securityContext:
          privileged: true
        volumeMounts:
        - name: cnibin
          mountPath: /host/opt/cni/bin
          mountPropagation: Bidirectional
      terminationGracePeriodSeconds: 10
      volumes:
      - name: cni
        hostPath:
          path: /etc/cni/net.d
      - name: cnibin
        hostPath:
          path: /opt/cni/bin
      - name: hostroot
        hostPath:
          path: /
      - name: multus-daemon-config
        configMap:
          name: multus-daemon-config
          items:
          - key: daemon-config.json
            path: daemon-config.json
      - name: host-run
        hostPath:
          path: /run
      - name: host-var-lib-cni-multus
        hostPath:
          path: /var/lib/cni/multus
      - name: host-var-lib-kubelet
        hostPath:
          path: /var/lib/kubelet
      - name: host-run-k8s-cni-cncf-io
        hostPath:
          path: /run/k8s.cni.cncf.io
      - name: host-run-netns
        hostPath:
          path: /run/netns/`

	piraeus = `---
# Source: piraeus/templates/rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: release-name-piraeus
  labels:
    helm.sh/chart: piraeus-2.5.1
    app.kubernetes.io/component: piraeus-operator
    app.kubernetes.io/name: piraeus-datastore
    app.kubernetes.io/instance: release-name
    app.kubernetes.io/version: "v2.5.1"
    app.kubernetes.io/managed-by: Helm
---
# Source: piraeus/templates/validating-webhook-configuration.yaml
# Check if the TLS secret already exists and initialize variables for later use at the top level



apiVersion: v1
kind: Secret
metadata:
  name: release-name-piraeus-tls
  labels:
    helm.sh/chart: piraeus-2.5.1
    app.kubernetes.io/component: piraeus-operator
    app.kubernetes.io/name: piraeus-datastore
    app.kubernetes.io/instance: release-name
    app.kubernetes.io/version: "v2.5.1"
    app.kubernetes.io/managed-by: Helm
type: kubernetes.io/tls
data:
  ca.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURmVENDQW1XZ0F3SUJBZ0lSQUsvYzU
---
# Source: piraeus/templates/config.yaml
# DO NOT EDIT; Automatically created by hack/copy-image-config-to-chart.sh
apiVersion: v1
kind: ConfigMap
metadata:
  name: release-name-piraeus-image-config
  labels:
    helm.sh/chart: piraeus-2.5.1
    app.kubernetes.io/component: piraeus-operator
    app.kubernetes.io/name: piraeus-datastore
    app.kubernetes.io/instance: release-name
    app.kubernetes.io/version: "v2.5.1"
    app.kubernetes.io/managed-by: Helm
data:
  0_piraeus_datastore_images.yaml: |
    ---
    # This is the configuration for default images used by piraeus-operator
    #
    # "base" is the default repository prefix to use.
    base: quay.io/piraeusdatastore
    # "components" is a mapping of image placeholders to actual image names with tag.
    # For example, the image name "linstor-controller" in the kustomize-resources will be replaced by:
    #   quay.io/piraeusdatastore/piraeus-server:v1.24.2
    components:
      linstor-controller:
        tag: v1.27.1
        image: piraeus-server
      linstor-satellite:
        # Pin with digest to ensure we pull the version with downgraded thin-send-recv
        tag: v1.27.1@sha256:26037f77d30d5487024e02a808d4ef913b93b745f2bb850cabc7f43a5359adff
        image: piraeus-server
      linstor-csi:
        tag: v1.6.0
        image: piraeus-csi
      drbd-reactor:
        tag: v1.4.0
        image: drbd-reactor
      ha-controller:
        tag: v1.2.1
        image: piraeus-ha-controller
      drbd-shutdown-guard:
        tag: v1.0.0
        image: drbd-shutdown-guard
      ktls-utils:
        tag: v0.10
        image: ktls-utils
      drbd-module-loader:
        tag: v9.2.9
        # The special "match" attribute is used to select an image based on the node's reported OS.
        # The operator will first check the k8s node's ".status.nodeInfo.osImage" field, and compare it against the list
        # here. If one matches, that specific image name will be used instead of the fallback image.
        image: drbd9-noble # Fallback image: chose a recent kernel, which can hopefully compile whatever config is actually in use
        match:
          - osImage: CentOS Linux 7
            image: drbd9-centos7
          - osImage: CentOS Linux 8
            image: drbd9-centos8
          - osImage: AlmaLinux 8
            image: drbd9-almalinux8
          - osImage: Red Hat Enterprise Linux CoreOS
            image: drbd9-almalinux8
          - osImage: AlmaLinux 9
            image: drbd9-almalinux9
          - osImage: Rocky Linux 8
            image: drbd9-almalinux8
          - osImage: Rocky Linux 9
            image: drbd9-almalinux9
          - osImage: Ubuntu 18\.04
            image: drbd9-bionic
          - osImage: Ubuntu 20\.04
            image: drbd9-focal
          - osImage: Ubuntu 22\.04
            image: drbd9-jammy
          - osImage: Ubuntu 24\.04
            image: drbd9-noble
          - osImage: Debian GNU/Linux 12
            image: drbd9-bookworm
          - osImage: Debian GNU/Linux 11
            image: drbd9-bullseye
          - osImage: Debian GNU/Linux 10
            image: drbd9-buster
  0_sig_storage_images.yaml: |
    ---
    base: registry.k8s.io/sig-storage
    components:
      csi-attacher:
        tag: v4.5.1
        image: csi-attacher
      csi-livenessprobe:
        tag: v2.12.0
        image: livenessprobe
      csi-provisioner:
        tag: v4.0.1
        image: csi-provisioner
      csi-snapshotter:
        tag: v7.0.2
        image: csi-snapshotter
      csi-resizer:
        tag: v1.10.1
        image: csi-resizer
      csi-external-health-monitor-controller:
        tag: v0.11.0
        image: csi-external-health-monitor-controller
      csi-node-driver-registrar:
        tag: v2.10.1
        image: csi-node-driver-registrar`

	specReplicasPath = "spec/replicas"
	deploymentKind   = "Deployment"
	appsGroup        = "apps"
	v1Version        = "v1"

	forceRecreateAnnotationKey = "projectsveltos.io/forceRecreate"
)

var _ = Describe("Applier utils", func() {
	It("customSplit returns all sections separated by ---", func() {
		sections, err := deployer.CustomSplit(multusData)
		Expect(err).To(BeNil())
		Expect(len(sections)).To(Equal(5))

		sections, err = deployer.CustomSplit("\n\n---\n")
		Expect(err).To(BeNil())
		Expect(len(sections)).To(Equal(0))

		sections, err = deployer.CustomSplit(piraeus)
		Expect(err).To(BeNil())
		Expect(len(sections)).To(Equal(3))

		multipleResources := `
apiVersion: v1
kind: Service
metadata:
labels:
  app: nats
  tailscale.com/proxy-class: default
annotations:
  tailscale.com/tailnet-fqdn: nats-cluster-1
name: nats-cluster-1
spec:
externalName: placeholder
type: ExternalName
---

apiVersion: v1
kind: Service
metadata:
labels:
  app: nats
  tailscale.com/proxy-class: default
annotations:
  tailscale.com/tailnet-fqdn: nats-cluster-2
name: nats-cluster-2
spec:
externalName: placeholder
type: ExternalName
---
`
		sections, err = deployer.CustomSplit(multipleResources)
		Expect(err).To(BeNil())
		Expect(len(sections)).To(Equal(2))
	})

	It("customSplit with array", func() {
		data := `  - |
    apiVersion: v1
    kind: Namespace
    metadata:
      name: fv-grfiheiz9d
  - |
    apiVersion: v1
    kind: Namespace1
    metadata:
      name: nginx
  - |
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      namespace: fv-grfiheiz9d
    spec:
      replicas: 2
      selector:
        matchLabels:
          app: nginx
      template:
        metadata:
          labels:
            app: nginx
        spec:
          containers:
          - image: nginx:latest
            name: nginx
            ports:
            - containerPort: 80`

		sections, err := deployer.CustomSplit(data)
		Expect(err).To(BeNil())
		Expect(len(sections)).To(Equal(3))
	})

	It("transformDriftExclusionsToPatches transforms DriftExclusions to Patches", func() {
		driftExclusions := []libsveltosv1beta1.DriftExclusion{
			{
				Paths: []string{specReplicasPath},
			},
			{
				Paths: []string{"spec/template/spec/containers[*]image"},
				Target: &libsveltosv1beta1.PatchSelector{
					Kind:    deploymentKind,
					Group:   appsGroup,
					Version: v1Version,
				},
			},
		}

		patches := deployer.TransformDriftExclusionsToPatches(driftExclusions)
		Expect(len(patches)).To(Equal(len(driftExclusions)))

		expectedPatch := libsveltosv1beta1.Patch{
			Patch: fmt.Sprintf(`- op: remove
  path: %s`, driftExclusions[0].Paths[0]),
		}

		Expect(patches).To(ContainElement(expectedPatch))

		expectedPatch = libsveltosv1beta1.Patch{
			Patch: fmt.Sprintf(`- op: remove
  path: %s`, driftExclusions[1].Paths[0]),
			Target: driftExclusions[1].Target,
		}
		Expect(patches).To(ContainElement(expectedPatch))
	})

	It("transformDriftExclusionsToPatches expands DriftExclusions paths to multiple to Patches", func() {
		driftExclusions := []libsveltosv1beta1.DriftExclusion{
			{
				Paths: []string{"spec/replicas", "metadata/labels"},
				Target: &libsveltosv1beta1.PatchSelector{
					Kind:    deploymentKind,
					Group:   appsGroup,
					Version: v1Version,
				},
			},
			{
				Paths: []string{"metadata/annotations", "spec.securityContext"},
				Target: &libsveltosv1beta1.PatchSelector{
					Kind:    "Pod",
					Group:   "",
					Version: v1Version,
				},
			},
		}

		patches := deployer.TransformDriftExclusionsToPatches(driftExclusions)
		Expect(len(patches)).To(Equal(2 * len(driftExclusions))) // each Paths has two entries

		expectedPatch := libsveltosv1beta1.Patch{
			Patch: fmt.Sprintf(`- op: remove
  path: %s`, driftExclusions[0].Paths[0]),
			Target: driftExclusions[0].Target,
		}

		Expect(patches).To(ContainElement(expectedPatch))

		expectedPatch = libsveltosv1beta1.Patch{
			Patch: fmt.Sprintf(`- op: remove
  path: %s`, driftExclusions[0].Paths[1]),
			Target: driftExclusions[0].Target,
		}

		Expect(patches).To(ContainElement(expectedPatch))

		expectedPatch = libsveltosv1beta1.Patch{
			Patch: fmt.Sprintf(`- op: remove
  path: %s`, driftExclusions[1].Paths[0]),
			Target: driftExclusions[1].Target,
		}
		Expect(patches).To(ContainElement(expectedPatch))

		expectedPatch = libsveltosv1beta1.Patch{
			Patch: fmt.Sprintf(`- op: remove
  path: %s`, driftExclusions[1].Paths[1]),
			Target: driftExclusions[1].Target,
		}
		Expect(patches).To(ContainElement(expectedPatch))
	})

	It("GetUnstructured with YAML/JSON/KYAML", func() {
		logger := textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))

		serviceYAML := `apiVersion: v1
kind: Service
metadata:
  creationTimestamp: "2025-05-09T21:14:40Z"
  labels:
    app: hostnames
  name: hostnames
  namespace: default
  resourceVersion: "37697"
  uid: "7aad616c-1686-4231-b32e-5ec68a738bba"
spec:
  clusterIP: "10.0.162.160"
  clusterIPs:
    - "10.0.162.160"
  internalTrafficPolicy: "Cluster"
  ipFamilies:
    - "IPv4"
  ipFamilyPolicy: "SingleStack"
  ports:
    - port: 80
      protocol: "TCP"
      targetPort: 9376
  selector:
    app: hostnames
  sessionAffinity: "None"
  type: "ClusterIP"
status:
  loadBalancer: {}`

		result, err := deployer.GetUnstructured([]byte(serviceYAML), logger)
		Expect(err).To(BeNil())
		Expect(len(result)).To(Equal(1))
		Expect(result[0].GetKind()).To(Equal("Service"))

		serviceJSON := `{
  "kind": "Service",
  "metadata": {
    "creationTimestamp": "2025-05-09T21:14:40Z",
    "labels": {
      "app": "hostnames"
    },
    "name": "hostnames",
    "namespace": "default",
    "resourceVersion": "37697",
    "uid": "7aad616c-1686-4231-b32e-5ec68a738bba"
  },
  "spec": {
    "clusterIP": "10.0.162.160",
    "clusterIPs": [
      "10.0.162.160"
    ],
    "internalTrafficPolicy": "Cluster",
    "ipFamilies": [
      "IPv4"
    ],
    "ipFamilyPolicy": "SingleStack",
    "ports": [
      {
        "port": 80,
        "protocol": "TCP",
        "targetPort": 9376
      }
    ],
    "selector": {
      "app": "hostnames"
    },
    "sessionAffinity": "None",
    "type": "ClusterIP"
  },
  "status": {
    "loadBalancer": {}
  }
}`

		result, err = deployer.GetUnstructured([]byte(serviceJSON), logger)
		Expect(err).To(BeNil())
		Expect(len(result)).To(Equal(1))
		Expect(result[0].GetKind()).To(Equal("Service"))

		serviceKYAML := `{
  apiVersion: "v1",
  kind: "Service",
  metadata: {
    creationTimestamp: "2025-05-09T21:14:40Z",
    labels: {
      app: "hostnames",
    },
    name: "hostnames",
    namespace: "default",
    resourceVersion: "37697",
    uid: "7aad616c-1686-4231-b32e-5ec68a738bba",
  },
  spec: {
    clusterIP: "10.0.162.160",
    clusterIPs: [
      "10.0.162.160",
    ],
    internalTrafficPolicy: "Cluster",
    ipFamilies: [
      "IPv4",
    ],
    ipFamilyPolicy: "SingleStack",
    ports: [{
      port: 80,
      protocol: "TCP",
      targetPort: 9376,
    }],
    selector: {
      app: "hostnames",
    },
    sessionAffinity: "None",
    type: "ClusterIP",
  },
  status: {
    loadBalancer: {},
  },
}`

		result, err = deployer.GetUnstructured([]byte(serviceKYAML), logger)
		Expect(err).To(BeNil())
		Expect(len(result)).To(Equal(1))
		Expect(result[0].GetKind()).To(Equal("Service"))
	})
})

var _ = Describe("requiresRecreate", func() {
	It("returns true for Invalid errors", func() {
		err := apierrors.NewInvalid(
			schema.GroupKind{Group: appsGroup, Kind: "Deployment"}, "foo", nil)
		Expect(deployer.RequiresRecreate(err)).To(BeTrue())
	})

	It("returns true for immutable field error text", func() {
		Expect(deployer.RequiresRecreate(fmt.Errorf("the field xyz is immutable"))).To(BeTrue())
		Expect(deployer.RequiresRecreate(fmt.Errorf("update rejected: immutable field detected"))).To(BeTrue())
	})

	It("returns false for unrelated errors", func() {
		Expect(deployer.RequiresRecreate(apierrors.NewNotFound(
			schema.GroupResource{Group: appsGroup, Resource: "deployments"}, "foo"))).To(BeFalse())
		Expect(deployer.RequiresRecreate(fmt.Errorf("connection refused"))).To(BeFalse())
	})

	It("returns false for nil", func() {
		Expect(deployer.RequiresRecreate(nil)).To(BeFalse())
	})
})

var _ = Describe("HasForceRecreateAnnotation", func() {
	It("returns true when the projectsveltos.io/forceRecreate annotation is set", func() {
		resource := &unstructured.Unstructured{}
		resource.SetAnnotations(map[string]string{forceRecreateAnnotationKey: ""})
		Expect(deployer.HasForceRecreateAnnotation(resource)).To(BeTrue())
	})

	It("returns false when the annotation is not set", func() {
		resource := &unstructured.Unstructured{}
		resource.SetAnnotations(map[string]string{"foo": "bar"})
		Expect(deployer.HasForceRecreateAnnotation(resource)).To(BeFalse())
	})

	It("returns false when there are no annotations", func() {
		resource := &unstructured.Unstructured{}
		Expect(deployer.HasForceRecreateAnnotation(resource)).To(BeFalse())
	})
})

var _ = Describe("GenerateErrorResourceReport", func() {
	It("returns a ResourceReport with Error action and the error message", func() {
		resource := &libsveltosv1beta1.Resource{
			Name:      randomString(),
			Namespace: randomString(),
			Kind:      "StatefulSet",
			Group:     appsGroup,
			Version:   "v1",
		}
		applyErr := fmt.Errorf("StatefulSet.apps %q is invalid: spec: Forbidden: updates to statefulset spec "+
			"for fields other than 'replicas' are forbidden", resource.Name)

		report := deployer.GenerateErrorResourceReport(resource, applyErr)

		Expect(report.Resource).To(Equal(*resource))
		Expect(report.Action).To(Equal(string(libsveltosv1beta1.ErrorResourceAction)))
		Expect(report.Message).To(Equal(applyErr.Error()))
	})
})

const (
	deploymentNoStrategyTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: main
        image: nginx:latest`

	deploymentRecreateStrategyTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: main
        image: nginx:latest`
)

var _ = Describe("UpdateResource force recreate", func() {
	It("recreates a Deployment when a strategy change is rejected by the API server and forceRecreate is set", func() {
		name := randomString()
		nsName := randomString()

		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		Expect(testEnv.Create(context.TODO(), ns)).To(Succeed())
		Expect(waitForObject(context.TODO(), testEnv.Client, ns)).To(Succeed())

		logger := textlogger.NewLogger(textlogger.NewConfig())

		initialYAML := fmt.Sprintf(deploymentNoStrategyTemplate, name, nsName, name, name)
		initialObj, err := k8s_utils.GetUnstructured([]byte(initialYAML))
		Expect(err).To(BeNil())

		dr, err := k8s_utils.GetDynamicResourceInterface(testEnv.Config, initialObj.GroupVersionKind(), nsName)
		Expect(err).To(BeNil())

		// Deployed the same way Sveltos would have before strategy was added: no strategy set,
		// so the API server defaults it to RollingUpdate with an explicit rollingUpdate value.
		_, err = deployer.UpdateResource(context.TODO(), dr, false, false, false,
			nil, initialObj, nil, logger)
		Expect(err).To(BeNil())

		Eventually(func() bool {
			current, getErr := dr.Get(context.TODO(), name, metav1.GetOptions{})
			if getErr != nil {
				return false
			}
			_, found, _ := unstructured.NestedMap(current.Object, "spec", "strategy", "rollingUpdate")
			return found
		}, time.Minute, 5*time.Second).Should(BeTrue())

		recreateYAML := fmt.Sprintf(deploymentRecreateStrategyTemplate, name, nsName, name, name)
		recreateObj, err := k8s_utils.GetUnstructured([]byte(recreateYAML))
		Expect(err).To(BeNil())

		// Without forceRecreate, the leftover rollingUpdate value conflicts with the new
		// strategy.type and the API server rejects the apply.
		_, err = deployer.UpdateResource(context.TODO(), dr, false, false, false,
			nil, recreateObj, nil, logger)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("rollingUpdate"))

		// With forceRecreate, the object is deleted and recreated to match the new manifest.
		updated, err := deployer.UpdateResource(context.TODO(), dr, false, false, true,
			nil, recreateObj, nil, logger)
		Expect(err).To(BeNil())
		Expect(updated).ToNot(BeNil())

		strategyType, _, _ := unstructured.NestedString(updated.Object, "spec", "strategy", "type")
		Expect(strategyType).To(Equal("Recreate"))

		_, found, _ := unstructured.NestedMap(updated.Object, "spec", "strategy", "rollingUpdate")
		Expect(found).To(BeFalse())
	})

	It("recreates a Deployment when forceRecreate is false but the object carries the forceRecreate annotation", func() {
		name := randomString()
		nsName := randomString()

		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		Expect(testEnv.Create(context.TODO(), ns)).To(Succeed())
		Expect(waitForObject(context.TODO(), testEnv.Client, ns)).To(Succeed())

		logger := textlogger.NewLogger(textlogger.NewConfig())

		initialYAML := fmt.Sprintf(deploymentNoStrategyTemplate, name, nsName, name, name)
		initialObj, err := k8s_utils.GetUnstructured([]byte(initialYAML))
		Expect(err).To(BeNil())

		dr, err := k8s_utils.GetDynamicResourceInterface(testEnv.Config, initialObj.GroupVersionKind(), nsName)
		Expect(err).To(BeNil())

		_, err = deployer.UpdateResource(context.TODO(), dr, false, false, false,
			nil, initialObj, nil, logger)
		Expect(err).To(BeNil())

		Eventually(func() bool {
			current, getErr := dr.Get(context.TODO(), name, metav1.GetOptions{})
			if getErr != nil {
				return false
			}
			_, found, _ := unstructured.NestedMap(current.Object, "spec", "strategy", "rollingUpdate")
			return found
		}, time.Minute, 5*time.Second).Should(BeTrue())

		recreateYAML := fmt.Sprintf(deploymentRecreateStrategyTemplate, name, nsName, name, name)
		recreateObj, err := k8s_utils.GetUnstructured([]byte(recreateYAML))
		Expect(err).To(BeNil())
		recreateObj.SetAnnotations(map[string]string{forceRecreateAnnotationKey: ""})

		// forceRecreate is false, but the resource-level annotation still triggers the
		// delete+recreate path.
		updated, err := deployer.UpdateResource(context.TODO(), dr, false, false, false,
			nil, recreateObj, nil, logger)
		Expect(err).To(BeNil())
		Expect(updated).ToNot(BeNil())

		strategyType, _, _ := unstructured.NestedString(updated.Object, "spec", "strategy", "type")
		Expect(strategyType).To(Equal("Recreate"))
	})
})
