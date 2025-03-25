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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
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

	It("transformDriftExclusionsToPatches transforms DriftExclusions to Patches", func() {
		driftExclusions := []libsveltosv1beta1.DriftExclusion{
			{
				Paths: []string{"spec/replicas"},
			},
			{
				Paths: []string{"spec/template/spec/containers[*]image"},
				Target: &libsveltosv1beta1.PatchSelector{
					Kind:    "Deployment",
					Group:   "apps",
					Version: "v1",
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
					Kind:    "Deployment",
					Group:   "apps",
					Version: "v1",
				},
			},
			{
				Paths: []string{"metadata/annotations", "spec.securityContext"},
				Target: &libsveltosv1beta1.PatchSelector{
					Kind:    "Pod",
					Group:   "",
					Version: "v1",
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
})
