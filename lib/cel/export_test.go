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

type ProgramCache = programCache

var (
	NewProgramCache = newProgramCache
)

// Get compiles expression (or reuses the cached program) and returns the compilation error, if any
func (c *programCache) Get(expression string) error {
	return c.get(expression).err
}

func (c *programCache) Len() int {
	return c.len()
}

// Contains reports whether expression is cached, without changing its position in the LRU order
func (c *programCache) Contains(expression string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.entries[expression]
	return ok
}

func (c *programCache) ExpressionBytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.expressionBytes
}

// ResetSharedCache empties the cache used by EvaluateRules
func ResetSharedCache() {
	sharedCache.reset()
}

func SharedCacheLen() int {
	return sharedCache.len()
}
