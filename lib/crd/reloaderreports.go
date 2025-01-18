// Generated by *go generate* - DO NOT EDIT
/*
Copyright 2022-23. projectsveltos.io. All rights reserved.

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
package crd

var ReloaderReportFile = "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_reloaderreports.lib.projectsveltos.io.yaml"
var ReloaderReportCRD = []byte(`apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.17.1
  name: reloaderreports.lib.projectsveltos.io
spec:
  group: lib.projectsveltos.io
  names:
    kind: ReloaderReport
    listKind: ReloaderReportList
    plural: reloaderreports
    singular: reloaderreport
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: ReloaderReport is the Schema for the ReloaderReport API
        properties:
          apiVersion:
            description: |-
              APIVersion defines the versioned schema of this representation of an object.
              Servers should convert recognized schemas to the latest internal value, and
              may reject unrecognized values.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
            type: string
          kind:
            description: |-
              Kind is a string value representing the REST resource this object represents.
              Servers may infer this from the endpoint the client submits requests to.
              Cannot be updated.
              In CamelCase.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
            type: string
          metadata:
            type: object
          spec:
            properties:
              clusterName:
                description: |-
                  ClusterName is the name of the Cluster this ReloaderReport
                  is for.
                type: string
              clusterNamespace:
                description: |-
                  ClusterNamespace is the namespace of the Cluster this
                  ReloaderReport is for.
                type: string
              clusterType:
                description: |-
                  ClusterType is the type of Cluster this ReloaderReport
                  is for.
                type: string
              resourcesToReload:
                description: |-
                  ResourcesToReload contains a list of resources that requires
                  rolling upgrade
                items:
                  description: |-
                    ReloaderInfo represents a resource that need to be reloaded
                    if any mounted ConfigMap/Secret changes.
                  properties:
                    kind:
                      description: 'Kind of the resource. Supported kinds are: Deployment
                        StatefulSet DaemonSet.'
                      enum:
                      - Deployment
                      - StatefulSet
                      - DaemonSet
                      type: string
                    name:
                      description: Name of the referenced resource.
                      minLength: 1
                      type: string
                    namespace:
                      description: Namespace of the referenced resource.
                      minLength: 1
                      type: string
                    value:
                      type: string
                  required:
                  - kind
                  - name
                  - namespace
                  type: object
                type: array
            required:
            - clusterName
            - clusterNamespace
            - clusterType
            type: object
          status:
            description: ReloaderReportStatus defines the observed state of ReloaderReport
            properties:
              phase:
                description: Phase represents the current phase of report.
                enum:
                - WaitingForDelivery
                - Delivering
                - Processed
                type: string
            type: object
        type: object
    served: true
    storage: false
    subresources:
      status: {}
  - name: v1beta1
    schema:
      openAPIV3Schema:
        description: ReloaderReport is the Schema for the ReloaderReport API
        properties:
          apiVersion:
            description: |-
              APIVersion defines the versioned schema of this representation of an object.
              Servers should convert recognized schemas to the latest internal value, and
              may reject unrecognized values.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
            type: string
          kind:
            description: |-
              Kind is a string value representing the REST resource this object represents.
              Servers may infer this from the endpoint the client submits requests to.
              Cannot be updated.
              In CamelCase.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
            type: string
          metadata:
            type: object
          spec:
            properties:
              clusterName:
                description: |-
                  ClusterName is the name of the Cluster this ReloaderReport
                  is for.
                type: string
              clusterNamespace:
                description: |-
                  ClusterNamespace is the namespace of the Cluster this
                  ReloaderReport is for.
                type: string
              clusterType:
                description: |-
                  ClusterType is the type of Cluster this ReloaderReport
                  is for.
                type: string
              resourcesToReload:
                description: |-
                  ResourcesToReload contains a list of resources that requires
                  rolling upgrade
                items:
                  description: |-
                    ReloaderInfo represents a resource that need to be reloaded
                    if any mounted ConfigMap/Secret changes.
                  properties:
                    kind:
                      description: 'Kind of the resource. Supported kinds are: Deployment
                        StatefulSet DaemonSet.'
                      enum:
                      - Deployment
                      - StatefulSet
                      - DaemonSet
                      type: string
                    name:
                      description: Name of the referenced resource.
                      minLength: 1
                      type: string
                    namespace:
                      description: Namespace of the referenced resource.
                      minLength: 1
                      type: string
                    value:
                      type: string
                  required:
                  - kind
                  - name
                  - namespace
                  type: object
                type: array
            required:
            - clusterName
            - clusterNamespace
            - clusterType
            type: object
          status:
            description: ReloaderReportStatus defines the observed state of ReloaderReport
            properties:
              phase:
                description: Phase represents the current phase of report.
                enum:
                - WaitingForDelivery
                - Delivering
                - Processed
                type: string
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
`)
