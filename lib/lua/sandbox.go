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

package lua

import (
	lua "github.com/yuin/gopher-lua"
)

type safeLibrary struct {
	name   string
	opener lua.LGFunction
}

var (
	// safeLibraries are the only standard libraries opened in a sandboxed state.
	// io, debug and channel are deliberately not part of it. The os library is
	// opened too but then pruned (see allowedOsFunctions).
	safeLibraries = []safeLibrary{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
		{lua.CoroutineLibName, lua.OpenCoroutine},
		{lua.OsLibName, lua.OpenOs},
	}

	// allowedOsFunctions are the only functions kept in the os table. They only
	// deal with time. Everything else (getenv, setenv, execute, exit, remove,
	// rename, tmpname, setlocale) reaches the host and is removed.
	allowedOsFunctions = map[string]bool{
		"time":     true,
		"date":     true,
		"clock":    true,
		"difftime": true,
	}

	// unsafeBaseFunctions are removed from the base library because they read
	// files from the host.
	unsafeBaseFunctions = []string{"dofile", "loadfile"}
)

// NewSandboxedState returns a Lua state for running scripts that are not fully
// trusted. Compared to lua.NewState it:
//   - does not open the io and debug libraries;
//   - keeps only time related functions in os (time, date, clock, difftime);
//   - removes dofile and loadfile;
//   - empties package.path so that require can only load preloaded modules
//     and never Lua files from the host file system.
//
// The caller must still call Close on the returned state.
func NewSandboxedState(opts ...lua.Options) *lua.LState {
	var options lua.Options
	if len(opts) > 0 {
		options = opts[0]
	}
	options.SkipOpenLibs = true

	l := lua.NewState(options)
	openSafeLibraries(l)
	restrictOsLibrary(l)
	removeUnsafeBaseFunctions(l)
	disableFileSystemRequire(l)

	return l
}

func openSafeLibraries(l *lua.LState) {
	for i := range safeLibraries {
		l.Push(l.NewFunction(safeLibraries[i].opener))
		l.Push(lua.LString(safeLibraries[i].name))
		l.Call(1, 0)
	}
}

// restrictOsLibrary removes from the os table every function not in
// allowedOsFunctions. A function added by a newer gopher-lua is removed by
// default.
func restrictOsLibrary(l *lua.LState) {
	osTable, ok := l.GetGlobal(lua.OsLibName).(*lua.LTable)
	if !ok {
		return
	}

	var toRemove []string
	osTable.ForEach(func(key, _ lua.LValue) {
		if name, isString := key.(lua.LString); isString && !allowedOsFunctions[string(name)] {
			toRemove = append(toRemove, string(name))
		}
	})

	for _, name := range toRemove {
		osTable.RawSetString(name, lua.LNil)
	}
}

func removeUnsafeBaseFunctions(l *lua.LState) {
	for _, name := range unsafeBaseFunctions {
		l.SetGlobal(name, lua.LNil)
	}
}

// disableFileSystemRequire makes the Lua file searcher of require find nothing.
// Modules registered with PreloadModule are still found.
func disableFileSystemRequire(l *lua.LState) {
	if pkg, ok := l.GetGlobal(lua.LoadLibName).(*lua.LTable); ok {
		pkg.RawSetString("path", lua.LString(""))
	}
}
