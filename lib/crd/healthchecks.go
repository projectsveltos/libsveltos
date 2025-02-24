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

var HealthCheckFile = "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_healthchecks.lib.projectsveltos.io.yaml"
var HealthCheckCRD = []byte(`apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.17.2
  name: healthchecks.lib.projectsveltos.io
spec:
  group: lib.projectsveltos.io
  names:
    kind: HealthCheck
    listKind: HealthCheckList
    plural: healthchecks
    singular: healthcheck
  scope: Cluster
  versions:
  - name: v1beta1
    schema:
      openAPIV3Schema:
        description: HealthCheck is the Schema for the HealthCheck API
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
            description: HealthCheckSpec defines the desired state of HealthCheck
            properties:
              collectResources:
                default: false
                description: |-
                  CollectResources indicates whether matching resources need
                  to be collected and added to HealthReport.
                type: boolean
              evaluateHealth:
                description: |-
                  The EvaluateHealth field specifies a Lua function responsible for evaluating the
                  health of the resources selected by resourceSelectors.
                  This function can assess the health of each resource independently or consider inter-resource relationships.
                  The function must be named *evaluate* and can access all objects identified by resourceSelectors using
                  the *resources* variable. It should return an array of structured instances, each containing the following fields:
                  - resource: The resource being evaluated
                  - healthStatus: The health status of the resource, which can be one of "Healthy", "Progressing", "Degraded", or "Suspended"
                  - message: An optional message providing additional information about the health status
                minLength: 1
                type: string
              resourceSelectors:
                description: ResourceSelectors identifies what resources to select
                  to evaluate health
                items:
                  description: ResourceSelector defines what resources are a match
                  properties:
                    evaluate:
                      description: |-
                        Evaluate contains a function "evaluate" in lua language.
                        The function will be passed one of the object selected based on
                        above criteria.
                        Must return struct with field "matching" representing whether
                        object is a match and an optional "message" field.
                      type: string
                    group:
                      description: Group of the resource deployed in the Cluster.
                      type: string
                    kind:
                      description: Kind of the resource deployed in the Cluster.
                      minLength: 1
                      type: string
                    labelFilters:
                      description: LabelFilters allows to filter resources based on
                        current labels.
                      items:
                        properties:
                          key:
                            description: Key is the label key
                            type: string
                          operation:
                            description: Operation is the comparison operation
                            enum:
                            - Equal
                            - Different
                            type: string
                          value:
                            description: Value is the label value
                            type: string
                        required:
                        - key
                        - operation
                        - value
                        type: object
                      type: array
                    name:
                      description: Name of the resource deployed in the  Cluster.
                      type: string
                    namespace:
                      description: |-
                        Namespace of the resource deployed in the  Cluster.
                        Empty for resources scoped at cluster level.
                        For namespaced resources, an empty string "" indicates all namespaces.
                      type: string
                    version:
                      description: Version of the resource deployed in the Cluster.
                      type: string
                  required:
                  - group
                  - kind
                  - version
                  type: object
                type: array
            required:
            - evaluateHealth
            - resourceSelectors
            type: object
        type: object
    served: true
    storage: true
`)
