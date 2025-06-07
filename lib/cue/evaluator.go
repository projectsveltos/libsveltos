/*
Copyright 2025. projectsveltos.io. All rights reserved.

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

package cue

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/projectsveltos/libsveltos/lib/logsettings"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

// EvaluateRules evaluates all rules against an unstructured object
// Returns true if any rule matches, along with the name of the matching rule
func EvaluateRules(obj *unstructured.Unstructured, rules []libsveltosv1beta1.CUERule,
	logger logr.Logger) (bool, error) {

	ctx := cuecontext.New()

	// Convert unstructured object to CUE value
	objValue := ctx.Encode(map[string]interface{}{
		"resource": obj.Object,
	})
	if objValue.Err() != nil {
		return false, objValue.Err()
	}

	for i := range rules {
		match, err := matchesRule(ctx, objValue, rules[i].Rule)
		if err != nil {
			logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to validate rule %s: %v",
				rules[i].Name, err))
			return false, err
		}

		if match {
			logger.V(logsettings.LogDebug).Info(fmt.Sprintf("object %s %s/%s matched rule %s",
				obj.GetKind(), obj.GetNamespace(), obj.GetName(), rules[i].Name))
			return true, nil
		}
	}

	return false, nil
}

func matchesRule(ctx *cue.Context, objValue cue.Value, rule string) (bool, error) {
	// Parse the constraint
	constraintValue := ctx.CompileString(rule)
	if constraintValue.Err() != nil {
		return false, constraintValue.Err()
	}

	// Unify the object with the constraint
	result := constraintValue.Unify(objValue)

	// Check if unification succeeded (no errors means it matches)
	return result.Err() == nil, nil
}
