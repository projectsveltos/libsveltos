// Generated by *go generate* - DO NOT EDIT
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
package crd

var HealthCheckReportFile = "../../config/crd/bases/lib.projectsveltos.io_healthcheckreports.yaml"
var HealthCheckReportCRD = []byte(`---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.10.0
  creationTimestamp: null
  name: healthcheckreports.lib.projectsveltos.io
spec:
  group: lib.projectsveltos.io
  names:
    kind: HealthCheckReport
    listKind: HealthCheckReportList
    plural: healthcheckreports
    singular: healthcheckreport
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: HealthCheckReport is the Schema for the HealthCheckReport API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            properties:
              clusterName:
                description: ClusterName is the name of the Cluster this HealthCheckReport
                  is for.
                type: string
              clusterNamespace:
                description: ClusterNamespace is the namespace of the Cluster this
                  HealthCheckReport is for.
                type: string
              clusterType:
                description: ClusterType is the type of Cluster this HealthCheckReport
                  is for.
                type: string
              healthCheckName:
                description: HealthName is the name of the HealthCheck instance this
                  report is for.
                type: string
              resourceStatuses:
                description: ResourceStatuses contains a list of resources with their
                  status
                items:
                  properties:
                    healthStatus:
                      description: HealthStatus is the health status of the object
                      enum:
                      - Healthy
                      - Progressing
                      - Degraded
                      - Suspended
                      type: string
                    message:
                      description: Message is an extra message for human consumption
                      type: string
                    objectRef:
                      description: ObjectRef for which status is reported
                      properties:
                        apiVersion:
                          description: API version of the referent.
                          type: string
                        fieldPath:
                          description: 'If referring to a piece of an object instead
                            of an entire object, this string should contain a valid
                            JSON/Go field access statement, such as desiredState.manifest.containers[2].
                            For example, if the object reference is to a container
                            within a pod, this would take on a value like: "spec.containers{name}"
                            (where "name" refers to the name of the container that
                            triggered the event) or if no container name is specified
                            "spec.containers[2]" (container with index 2 in this pod).
                            This syntax is chosen only to have some well-defined way
                            of referencing a part of an object. TODO: this design
                            is not final and this field is subject to change in the
                            future.'
                          type: string
                        kind:
                          description: 'Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
                          type: string
                        name:
                          description: 'Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names'
                          type: string
                        namespace:
                          description: 'Namespace of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/'
                          type: string
                        resourceVersion:
                          description: 'Specific resourceVersion to which this reference
                            is made, if any. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency'
                          type: string
                        uid:
                          description: 'UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids'
                          type: string
                      type: object
                      x-kubernetes-map-type: atomic
                  required:
                  - healthStatus
                  - objectRef
                  type: object
                type: array
            required:
            - clusterName
            - clusterNamespace
            - clusterType
            - healthCheckName
            type: object
          status:
            description: HealthCheckReportStatus defines the observed state of HealthCheckReport
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
