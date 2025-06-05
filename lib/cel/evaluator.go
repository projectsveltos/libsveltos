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

	"github.com/go-logr/logr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/logsettings"
)

// EvaluateRules evaluates all rules for a specific GVK against an unstructured object
// Returns true if any rule matches, along with the name of the matching rule
func EvaluateRules(gvk schema.GroupVersionKind, resource *unstructured.Unstructured,
	rules []libsveltosv1beta1.CELRule, logger logr.Logger) (matched bool, err error) {

	// Evaluate each match rule
	for i := range rules {
		rule := &rules[i]
		logger.V(logsettings.LogDebug).Info("evaluate match rule %s", rule.Name)
		matched, err = evaluateRule(rule.Rule, resource, logger)
		if err != nil {
			klog.Warningf("Failed to evaluate rule %s for %s: %v", rule.Name, gvk.String(), err)
			continue
		}

		if matched {
			return true, nil
		}
	}

	return false, nil
}

// evaluateRule evaluates a single CEL expression against an object
func evaluateRule(expression string, resource *unstructured.Unstructured,
	logger logr.Logger) (bool, error) {

	env, err := cel.NewEnv(
		cel.Variable("resource", cel.DynType),
	)
	if err != nil {
		err = fmt.Errorf("failed to create CEL environment: %w", err)
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, err
	}

	ast, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		err = fmt.Errorf("failed to parse CEL expression: %w", issues.Err())
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, err
	}

	checked, issues := env.Check(ast)
	if issues != nil && issues.Err() != nil {
		err = fmt.Errorf("failed to check CEL expression: %w", issues.Err())
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, err
	}

	program, err := env.Program(checked)
	if err != nil {
		err = fmt.Errorf("failed to create CEL program: %w", err)
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, err
	}

	result, _, err := program.Eval(map[string]interface{}{
		"resource": resource.Object,
	})
	if err != nil {
		err = fmt.Errorf("failed to evaluate CEL expression: %w", err)
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	if result.Type() != types.BoolType {
		err = fmt.Errorf("expected boolean result, got: %v", result.Type())
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("evaluateMatchRule failed: %v", err))
		return false, err
	}

	return result.Value().(bool), nil
}
