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

package lua_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lua "github.com/yuin/gopher-lua"

	lua_modules "github.com/projectsveltos/libsveltos/lib/lua"
)

var _ = Describe("NewSandboxedState", func() {
	var l *lua.LState

	BeforeEach(func() {
		l = lua_modules.NewSandboxedState()
		lua_modules.LoadModulesAndRegisterMethods(l)
	})

	AfterEach(func() {
		l.Close()
	})

	It("does not expose the io and debug libraries", func() {
		Expect(l.GetGlobal("io")).To(Equal(lua.LNil))
		Expect(l.GetGlobal("debug")).To(Equal(lua.LNil))
		Expect(l.DoString(`local f = io.open("/etc/hostname", "r")`)).ToNot(Succeed())
	})

	It("keeps only the time functions in os", func() {
		osTable, ok := l.GetGlobal("os").(*lua.LTable)
		Expect(ok).To(BeTrue())

		for _, name := range []string{"time", "date", "clock", "difftime"} {
			Expect(osTable.RawGetString(name).Type()).To(Equal(lua.LTFunction), name)
		}

		for _, name := range []string{"getenv", "setenv", "execute", "exit", "remove", "rename", "tmpname", "setlocale"} {
			Expect(osTable.RawGetString(name)).To(Equal(lua.LNil), name)
		}
	})

	It("cannot read the environment", func() {
		Expect(os.Setenv("SVELTOS_LUA_SANDBOX_TEST", "secret")).To(Succeed())
		defer os.Unsetenv("SVELTOS_LUA_SANDBOX_TEST")

		Expect(l.DoString(`leaked = os.getenv("SVELTOS_LUA_SANDBOX_TEST")`)).ToNot(Succeed())
		Expect(l.GetGlobal("leaked")).To(Equal(lua.LNil))
	})

	It("removes dofile and loadfile", func() {
		Expect(l.GetGlobal("dofile")).To(Equal(lua.LNil))
		Expect(l.GetGlobal("loadfile")).To(Equal(lua.LNil))
	})

	It("does not require Lua files from the file system", func() {
		// The default package.path contains ./?.lua: a module in the working
		// directory is loadable by a standard state but not by the sandboxed one.
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "hostmodule.lua"), []byte(`return {}`), 0o600)).To(Succeed())

		workingDir, err := os.Getwd()
		Expect(err).To(BeNil())
		Expect(os.Chdir(dir)).To(Succeed())
		defer func() { Expect(os.Chdir(workingDir)).To(Succeed()) }()

		standard := lua.NewState()
		defer standard.Close()
		Expect(standard.DoString(`require("hostmodule")`)).To(Succeed())

		Expect(l.DoString(`require("hostmodule")`)).ToNot(Succeed())
	})

	It("still supports time functions", func() {
		Expect(l.DoString(`
			local currentTime = os.time(os.date("!*t"))
			assert(type(currentTime) == "number")
			assert(os.difftime(currentTime, currentTime) == 0)
			assert(type(os.clock()) == "number")
		`)).To(Succeed())
	})

	It("still supports the standard libraries and Sveltos modules", func() {
		Expect(l.DoString(`
			local json = require("json")
			local strings = require("strings")
			assert(string.upper("a") == "A")
			assert(math.floor(1.5) == 1)
			assert(#table.concat({"a", "b"}) == 2)
			assert(base64Decode(base64Encode("x")) == "x")
			assert(type(json) == "table" and type(strings) == "table")
		`)).To(Succeed())
	})

	It("honors the options given by the caller", func() {
		custom := lua_modules.NewSandboxedState(lua.Options{CallStackSize: 64, RegistrySize: 1024})
		defer custom.Close()

		Expect(custom.GetGlobal("io")).To(Equal(lua.LNil))
		Expect(custom.DoString(`return 1`)).To(Succeed())
	})
})
