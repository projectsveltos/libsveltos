/*
Copyright 2024. projectsveltos.io. All rights reserved.

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

package lua

import (
	"encoding/base64"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"

	luajson "github.com/projectsveltos/lua-utils/glua-json"
	luarunes "github.com/projectsveltos/lua-utils/glua-runes"
	luastrings "github.com/projectsveltos/lua-utils/glua-strings"
)

const (
	LuaTableError = "lua script output is not a lua table"
	LuaBoolError  = "lua script output is not a lua bool"
)

func LoadModulesAndRegisterMethods(l *lua.LState) {
	l.PreloadModule("json", luajson.Loader)
	l.PreloadModule("strings", luastrings.Loader)
	l.PreloadModule("runes", luarunes.Loader)

	l.SetGlobal("base64Encode", l.NewFunction(base64Encode))
	l.SetGlobal("base64Decode", l.NewFunction(base64Decode))
	registerGetLabel(l)
	registerGetAnnotation(l)
	registerGetLuaResource(l)
}

// MapToTable converts a Go map to a lua table
// credit to: https://github.com/yuin/gopher-lua/issues/160#issuecomment-447608033
func MapToTable(m map[string]interface{}) *lua.LTable {
	// Main table pointer
	resultTable := &lua.LTable{}

	// Loop map
	for key, element := range m {
		switch element := element.(type) {
		case float64:
			resultTable.RawSetString(key, lua.LNumber(element))
		case int64:
			resultTable.RawSetString(key, lua.LNumber(element))
		case string:
			resultTable.RawSetString(key, lua.LString(element))
		case bool:
			resultTable.RawSetString(key, lua.LBool(element))
		case []byte:
			resultTable.RawSetString(key, lua.LString(string(element)))
		case map[string]interface{}:

			// Get table from map
			tble := MapToTable(element)

			resultTable.RawSetString(key, tble)

		case time.Time:
			resultTable.RawSetString(key, lua.LNumber(element.Unix()))

		case []map[string]interface{}:

			// Create slice table
			sliceTable := &lua.LTable{}

			// Loop element
			for _, s := range element {
				// Get table from map
				tble := MapToTable(s)

				sliceTable.Append(tble)
			}

			// Set slice table
			resultTable.RawSetString(key, sliceTable)

		case []interface{}:

			// Create slice table
			sliceTable := &lua.LTable{}

			// Loop interface slice
			for _, s := range element {
				// Switch interface type
				switch s := s.(type) {
				case map[string]interface{}:

					// Convert map to table
					t := MapToTable(s)

					// Append result
					sliceTable.Append(t)

				case float64:

					// Append result as number
					sliceTable.Append(lua.LNumber(s))

				case string:

					// Append result as string
					sliceTable.Append(lua.LString(s))

				case bool:

					// Append result as bool
					sliceTable.Append(lua.LBool(s))
				}
			}

			// Append to main table
			resultTable.RawSetString(key, sliceTable)
		}
	}

	return resultTable
}

// ToGoValue converts the given LValue to a Go object.
// Credit to: https://github.com/yuin/gluamapper/blob/master/gluamapper.go
func ToGoValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn == 0 { // table
			ret := make(map[string]interface{})
			v.ForEach(func(key, value lua.LValue) {
				keystr := fmt.Sprint(ToGoValue(key))
				ret[keystr] = ToGoValue(value)
			})
			return ret
		} else { // array
			ret := make([]interface{}, 0, maxn)
			for i := 1; i <= maxn; i++ {
				ret = append(ret, ToGoValue(v.RawGetInt(i)))
			}
			return ret
		}
	default:
		return v
	}
}

func base64Encode(l *lua.LState) int {
	if l == nil {
		l.Push(lua.LNil)
		l.Push(lua.LString("nil state passed"))
		return 2 //nolint: mnd // number of return values
	}

	str := l.CheckString(1)
	if str == "" {
		l.Push(lua.LNil)
		l.Push(lua.LString("provided string is empty"))
		return 2 //nolint: mnd // number of return values
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(str))
	l.Push(lua.LString(encoded))
	return 1
}

func base64Decode(l *lua.LState) int {
	if l == nil {
		l.Push(lua.LNil)
		l.Push(lua.LString("nil state passed"))
		return 2 //nolint: mnd // number of return values
	}

	str := l.CheckString(1)
	if str == "" {
		l.Push(lua.LNil)
		l.Push(lua.LString("provided string is empty"))
		return 2 //nolint: mnd // number of return values
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString("deconding failed"))
		return 2 //nolint: mnd // number of return values
	}

	l.Push(lua.LString(decodedBytes))
	return 1
}

func getLabel(table *lua.LTable, key string) (string, bool) {
	if table == nil {
		return "", false
	}

	metadata := table.RawGetString("metadata")
	if metadata == lua.LNil {
		return "", false
	}

	metadataTable, ok := metadata.(*lua.LTable)
	if !ok {
		return "", false
	}

	labels := metadataTable.RawGetString("labels")
	if labels == lua.LNil {
		return "", false
	}

	labelsTable, ok := labels.(*lua.LTable)
	if !ok {
		return "", false
	}

	labelValue := labelsTable.RawGetString(key)
	if labelValue == lua.LNil {
		return "", false // Key not found
	}

	labelStr, ok := labelValue.(lua.LString)
	if !ok {
		return "", false // Value is not a string
	}

	return string(labelStr), true
}

func getAnnotation(table *lua.LTable, key string) (string, bool) {
	if table == nil {
		return "", false
	}

	metadata := table.RawGetString("metadata")
	if metadata == lua.LNil {
		return "", false
	}

	metadataTable, ok := metadata.(*lua.LTable)
	if !ok {
		return "", false
	}

	labels := metadataTable.RawGetString("annotations")
	if labels == lua.LNil {
		return "", false
	}

	labelsTable, ok := labels.(*lua.LTable)
	if !ok {
		return "", false
	}

	labelValue := labelsTable.RawGetString(key)
	if labelValue == lua.LNil {
		return "", false // Key not found
	}

	labelStr, ok := labelValue.(lua.LString)
	if !ok {
		return "", false // Value is not a string
	}

	return string(labelStr), true
}

func getLuaResource(resources *lua.LTable, key string) (lua.LValue, bool) {
	if resources == nil {
		return lua.LNil, false
	}

	resource := resources.RawGetString(key)
	if resource == lua.LNil {
		return lua.LNil, false
	}

	return resource, true
}

func registerGetLabel(l *lua.LState) int {
	// Create a Go closure that adapts your getLabel function
	fn := func(L *lua.LState) int {
		// Get the arguments from Lua
		if L.GetTop() != 2 { //nolint: mnd // number of expected arg
			L.Push(lua.LNil)
			L.Push(lua.LString("expected 2 arguments (table, key)"))
			return 2 //nolint: mnd // number of return values
		}

		table := L.CheckTable(1)

		key := L.CheckString(2) //nolint: mnd // second arg

		// Call your getLabel function
		label, ok := getLabel(table, key)

		// Push the results back to Lua
		if ok {
			L.Push(lua.LString(label))
			return 1 // 1 result: the label string
		} else {
			L.Push(lua.LNil)
			return 1 // 1 result: nil (label not found)
		}
	}

	// Register the closure as a Lua function
	l.SetGlobal("getLabel", l.NewFunction(fn))
	return 0 // No return values from the registration function itself
}

func registerGetAnnotation(l *lua.LState) int {
	// Create a Go closure that adapts your getAnnotation function
	fn := func(L *lua.LState) int {
		// Get the arguments from Lua
		if L.GetTop() != 2 { //nolint: mnd // number of expected arg
			L.Push(lua.LNil)
			L.Push(lua.LString("expected 2 arguments (table, key)"))
			return 2 //nolint: mnd // number of return values
		}

		table := L.CheckTable(1)

		key := L.CheckString(2) //nolint: mnd // second arg

		// Call your getAnnotation function
		label, ok := getAnnotation(table, key)

		// Push the results back to Lua
		if ok {
			L.Push(lua.LString(label))
			return 1 // 1 result: the label string
		} else {
			L.Push(lua.LNil)
			return 1 // 1 result: nil (label not found)
		}
	}

	// Register the closure as a Lua function
	l.SetGlobal("getAnnotation", l.NewFunction(fn))
	return 0 // No return values from the registration function itself
}

func registerGetLuaResource(l *lua.LState) int {
	// Create a Go closure that adapts your getLabel function
	fn := func(L *lua.LState) int {
		// Get the arguments from Lua
		if L.GetTop() != 2 { //nolint: mnd // number of expected arg
			L.Push(lua.LNil)
			L.Push(lua.LString("expected 2 arguments (table, key)"))
			return 2 //nolint: mnd // number of return values
		}

		table := L.CheckTable(1)

		key := L.CheckString(2) //nolint: mnd // second arg

		// Call your getLuaResource function
		resource, ok := getLuaResource(table, key)

		// Push the results back to Lua
		if ok {
			L.Push(resource)
			return 1 // 1 result: the resorce
		} else {
			L.Push(lua.LNil)
			return 1 // 1 result: nil (resource not found)
		}
	}

	// Register the closure as a Lua function
	l.SetGlobal("getResource", l.NewFunction(fn))
	return 0 // No return values from the registration function itself
}
