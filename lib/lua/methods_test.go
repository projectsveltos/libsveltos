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

package lua_test

import (
	"testing"

	lua "github.com/yuin/gopher-lua"

	lua_modules "github.com/projectsveltos/libsveltos/lib/lua"
)

func TestLoadModulesAndRegisterMethods(t *testing.T) {
	l := lua.NewState()
	defer l.Close()

	lua_modules.LoadModulesAndRegisterMethods(l)

	expectedGlobals := []string{
		"base64Encode",
		"base64Decode",
		"getLabel",
		"getAnnotation",
		"getResource",
	}

	for _, name := range expectedGlobals {
		if glob := l.GetGlobal(name); glob.Type() != lua.LTFunction {
			t.Errorf("Expected global function '%s' to be registered, but it wasn't", name)
		}
	}

	modules := []string{
		"json",
		"strings",
		"runes",
	}

	for _, module := range modules {
		if err := l.DoString(`require "` + module + `"`); err != nil {
			t.Errorf("Module '%s' failed to load: %v", module, err)
		}
	}
}
