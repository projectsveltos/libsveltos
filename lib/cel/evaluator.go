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

package cel

import (
	"fmt"

	"cel.dev/cel-go/common/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

// EvaluateRules evaluates all rules against an unstructured object
// Returns true if any rule matches
func EvaluateRules(resource *unstructured.Unstructured,
	rules []libsveltosv1beta1.CELRule, logger logr.Logger) (matched bool, err error) {

	// Evaluate each match rule
	for i := range rules {
		rule := &rules[i]
		logger.V(logs.LogDebug).Info(fmt.Sprintf("evaluate match rule %s", rule.Name))
		matched, err = evaluateRule(rule.Rule, resource, logger)
		if err != nil {
			logger.V(logs.LogInfo).Info(fmt.Sprintf("Failed to evaluate rule %s %s/%s for %s: %v",
				rule.Name, resource.GetKind(), resource.GetNamespace(), resource.GetName(), err))
			continue
		}

		if matched {
			return true, nil
		}
	}

	return false, nil
}

// evaluateRule evaluates a single CEL expression against an object. The expression is compiled the
// first time it is seen and the compiled program is reused for every following evaluation.
func evaluateRule(expression string, resource *unstructured.Unstructured,
	logger logr.Logger) (bool, error) {

	rule := sharedCache.get(expression)
	if rule.err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", rule.err))
		return false, rule.err
	}

	result, _, err := rule.program.Eval(map[string]interface{}{
		"resource": resource.Object,
	})
	if err != nil {
		err = fmt.Errorf("failed to evaluate CEL expression: %w", err)
		logger.V(logs.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	if result.Type() != types.BoolType {
		err = fmt.Errorf("expected boolean result, got: %v", result.Type())
		logger.V(logs.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, err
	}

	return result.Value().(bool), nil
}
