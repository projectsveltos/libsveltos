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

package randutils

import (
	"math/rand"
)

const (
	charSet = "0123456789abcdefghijklmnopqrstuvwxyz"
)

// RandomString returns a goroutine-safe random alphanumeric string of length n.
// It uses the package-level math/rand source which is goroutine-safe.
func RandomString(n int) string {
	result := make([]byte, n)
	for i := range result {
		result[i] = charSet[rand.Intn(len(charSet))] //nolint:gosec // used just for name generation
	}
	return string(result)
}
