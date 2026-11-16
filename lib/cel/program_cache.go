/*
Copyright 2026. projectsveltos.io. All rights reserved.

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
	"container/list"
	"fmt"
	"sync"

	"cel.dev/cel-go/cel"
)

const (
	// maxCachedPrograms is the maximum number of distinct expressions kept compiled.
	maxCachedPrograms = 512

	// maxCachedExpressionBytes is the maximum sum of the lengths of the cached expressions.
	// It protects against a few very large rules filling the cache: a compiled program is
	// 7 to 16 times bigger than the text it was built from (measured), so this bounds the memory
	// held by the cache to a few tens of MB in the worst case.
	maxCachedExpressionBytes = 1024 * 1024
)

// compiledRule is the result of compiling one expression. A failed compilation is cached too, so
// an invalid expression is not parsed again for every object it is evaluated against.
type compiledRule struct {
	expression string
	program    cel.Program
	err        error
}

// programCache keeps compiled CEL programs, keyed by the text of the expression. The key is the
// content, so an edited rule is simply a different key: there is nothing to invalidate. When the
// cache is full the least recently used expression is dropped.
type programCache struct {
	mu sync.Mutex

	maxEntries         int
	maxExpressionBytes int

	// order holds *compiledRule. Front is the most recently used.
	order *list.List
	// entries maps an expression to its element in order.
	entries map[string]*list.Element
	// expressionBytes is the sum of the lengths of all cached expressions.
	expressionBytes int
}

func newProgramCache(maxEntries, maxExpressionBytes int) *programCache {
	return &programCache{
		maxEntries:         maxEntries,
		maxExpressionBytes: maxExpressionBytes,
		order:              list.New(),
		entries:            make(map[string]*list.Element),
	}
}

// get returns the compiled rule for expression, compiling it if it is not cached yet.
// Compilation happens outside the lock: if two goroutines miss on the same expression at the same
// time both compile it, and the second result is discarded. Programs are safe for concurrent use.
func (c *programCache) get(expression string) *compiledRule {
	if rule := c.lookup(expression); rule != nil {
		return rule
	}

	program, err := compile(expression)
	rule := &compiledRule{expression: expression, program: program, err: err}

	return c.store(rule)
}

func (c *programCache) lookup(expression string) *compiledRule {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.entries[expression]
	if !ok {
		return nil
	}

	c.order.MoveToFront(element)

	return element.Value.(*compiledRule)
}

// store adds rule to the cache and returns the rule to use. If another goroutine stored the same
// expression first, that one is returned.
func (c *programCache) store(rule *compiledRule) *compiledRule {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.entries[rule.expression]; ok {
		c.order.MoveToFront(element)
		return element.Value.(*compiledRule)
	}

	// An expression larger than the whole budget is used but never cached
	if len(rule.expression) > c.maxExpressionBytes {
		return rule
	}

	c.entries[rule.expression] = c.order.PushFront(rule)
	c.expressionBytes += len(rule.expression)

	for c.order.Len() > c.maxEntries || c.expressionBytes > c.maxExpressionBytes {
		c.evictLeastRecentlyUsed()
	}

	return rule
}

func (c *programCache) evictLeastRecentlyUsed() {
	element := c.order.Back()
	rule := c.order.Remove(element).(*compiledRule)
	delete(c.entries, rule.expression)
	c.expressionBytes -= len(rule.expression)
}

func (c *programCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}

func (c *programCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.order.Init()
	c.entries = make(map[string]*list.Element)
	c.expressionBytes = 0
}

var (
	// The environment is the same for every rule: a single "resource" variable. It is safe for
	// concurrent use, so it is built once.
	envOnce      sync.Once
	sharedEnv    *cel.Env
	errSharedEnv error

	sharedCache = newProgramCache(maxCachedPrograms, maxCachedExpressionBytes)
)

func getEnv() (*cel.Env, error) {
	envOnce.Do(func() {
		sharedEnv, errSharedEnv = cel.NewEnv(
			cel.Variable("resource", cel.DynType),
		)
		if errSharedEnv != nil {
			errSharedEnv = fmt.Errorf("failed to create CEL environment: %w", errSharedEnv)
		}
	})

	return sharedEnv, errSharedEnv
}

// compile parses, type-checks and plans a CEL expression
func compile(expression string) (cel.Program, error) {
	env, err := getEnv()
	if err != nil {
		return nil, err
	}

	ast, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to parse CEL expression: %w", issues.Err())
	}

	checked, issues := env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to check CEL expression: %w", issues.Err())
	}

	program, err := env.Program(checked)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	return program, nil
}
