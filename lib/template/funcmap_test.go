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

package template_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/projectsveltos/libsveltos/lib/template"
)

const (
	testKey   = "foo"
	testValue = "bar"
)

var _ = Describe("FuncMap", func() {
	funcMap := template.ExtraFuncMap()

	It("registers all the expected function names", func() {
		for _, name := range []string{
			"toToml", "fromToml", "toYaml", "fromYaml", "fromYamlArray",
			"toJson", "fromJson", "fromJsonArray",
		} {
			Expect(funcMap).To(HaveKey(name))
		}
	})

	It("toYaml marshals a value and fromYaml parses it back", func() {
		toYaml := funcMap["toYaml"].(func(interface{}) string)
		fromYaml := funcMap["fromYaml"].(func(string) map[string]interface{})

		out := toYaml(map[string]interface{}{testKey: testValue})
		Expect(out).To(Equal("foo: bar"))

		parsed := fromYaml(out)
		Expect(parsed[testKey]).To(Equal(testValue))
	})

	It("fromYaml records the parse error instead of failing", func() {
		fromYaml := funcMap["fromYaml"].(func(string) map[string]interface{})
		result := fromYaml("not: valid: yaml: [")
		Expect(result).To(HaveKey("Error"))
	})

	It("fromYamlArray parses a YAML array and records the parse error otherwise", func() {
		fromYamlArray := funcMap["fromYamlArray"].(func(string) []interface{})

		result := fromYamlArray("- a\n- b\n")
		Expect(result).To(Equal([]interface{}{"a", "b"}))

		errResult := fromYamlArray("not: valid: yaml: [")
		Expect(errResult).To(HaveLen(1))
	})

	It("toToml marshals a value and fromToml parses it back", func() {
		toToml := funcMap["toToml"].(func(interface{}) string)
		fromToml := funcMap["fromToml"].(func(string) map[string]interface{})

		out := toToml(map[string]interface{}{testKey: testValue})
		parsed := fromToml(out)
		Expect(parsed[testKey]).To(Equal(testValue))
	})

	It("fromToml records the parse error instead of failing", func() {
		fromToml := funcMap["fromToml"].(func(string) map[string]interface{})
		result := fromToml("not valid toml [[[")
		Expect(result).To(HaveKey("Error"))
	})

	It("toJson marshals a value and fromJson parses it back", func() {
		toJson := funcMap["toJson"].(func(interface{}) string)
		fromJson := funcMap["fromJson"].(func(string) map[string]interface{})

		out := toJson(map[string]interface{}{testKey: testValue})
		Expect(out).To(Equal(`{"foo":"bar"}`))

		parsed := fromJson(out)
		Expect(parsed[testKey]).To(Equal(testValue))
	})

	It("toJson returns an empty string when the value cannot be marshaled", func() {
		toJson := funcMap["toJson"].(func(interface{}) string)
		Expect(toJson(func() {})).To(Equal(""))
	})

	It("fromJson records the parse error instead of failing", func() {
		fromJson := funcMap["fromJson"].(func(string) map[string]interface{})
		result := fromJson("{not valid json")
		Expect(result).To(HaveKey("Error"))
	})

	It("fromJsonArray parses a JSON array and records the parse error otherwise", func() {
		fromJsonArray := funcMap["fromJsonArray"].(func(string) []interface{})

		result := fromJsonArray(`["a","b"]`)
		Expect(result).To(Equal([]interface{}{"a", "b"}))

		errResult := fromJsonArray("[not valid json")
		Expect(errResult).To(HaveLen(1))
	})
})
