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
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
	lua "github.com/yuin/gopher-lua"

	lua_modules "github.com/projectsveltos/libsveltos/lib/lua"
)

const nestedKey = "nested"

// referenceMapToTable is the original implementation (no pre-sizing). It is kept here as the
// behavioral reference: MapToTable must produce an equivalent table for any input.
func referenceMapToTable(m map[string]any) *lua.LTable {
	resultTable := &lua.LTable{}

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
		case map[string]any:
			resultTable.RawSetString(key, referenceMapToTable(element))
		case time.Time:
			resultTable.RawSetString(key, lua.LNumber(element.Unix()))
		case []map[string]any:
			sliceTable := &lua.LTable{}
			for _, s := range element {
				sliceTable.Append(referenceMapToTable(s))
			}
			resultTable.RawSetString(key, sliceTable)
		case []any:
			sliceTable := &lua.LTable{}
			for _, s := range element {
				switch s := s.(type) {
				case map[string]any:
					sliceTable.Append(referenceMapToTable(s))
				case float64:
					sliceTable.Append(lua.LNumber(s))
				case string:
					sliceTable.Append(lua.LString(s))
				case bool:
					sliceTable.Append(lua.LBool(s))
				}
			}
			resultTable.RawSetString(key, sliceTable)
		}
	}

	return resultTable
}

func readHeavyPod() (map[string]any, error) {
	data, err := os.ReadFile("testdata/heavy_pod.json")
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func loadHeavyPod() map[string]any {
	m, err := readHeavyPod()
	Expect(err).ToNot(HaveOccurred())

	return m
}

// bytesPerCall returns the average number of heap bytes allocated by one call of f.
func bytesPerCall(f func()) uint64 {
	const runs = 200

	f() // warm up

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for i := 0; i < runs; i++ {
		f()
	}
	runtime.ReadMemStats(&after)

	return (after.TotalAlloc - before.TotalAlloc) / runs
}

var _ = Describe("MapToTable", func() {
	// Inputs cover the supported types, including the ones that are silently skipped today
	// (nested slices, nil, int) so that behavior stays identical.
	DescribeTable("produces the same table as the reference implementation",
		func(in map[string]any) {
			got := lua_modules.ToGoValue(lua_modules.MapToTable(in))
			want := lua_modules.ToGoValue(referenceMapToTable(in))
			Expect(got).To(Equal(want))
		},
		Entry("empty map", map[string]any{}),
		Entry("scalars", map[string]any{
			"f": float64(1.5), "i": int64(7), "s": "x", "b": true, "bytes": []byte("abc"),
			"t": time.Unix(1700000000, 0),
		}),
		Entry("slices", map[string]any{
			"maps":    []map[string]any{{"a": "b"}, {}},
			"any":     []any{map[string]any{"k": "v"}, float64(2), "s", true, nil, int64(3), []any{nestedKey}},
			"empty":   []any{},
			nestedKey: map[string]any{"deep": map[string]any{"deeper": []any{"x", map[string]any{"y": float64(1)}}}},
		}),
		Entry("unsupported types are skipped", map[string]any{
			"nil": nil, "int": 3, "ints": []int{1}, "strs": []string{"a"},
		}),
		Entry("a real pod", loadHeavyPod()),
	)

	It("allocates far less than the version without pre-sizing", func() {
		pod := loadHeavyPod()

		reference := bytesPerCall(func() { _ = referenceMapToTable(pod) })
		current := bytesPerCall(func() { _ = lua_modules.MapToTable(pod) })

		By(fmt.Sprintf("reference: %d B per call, current: %d B per call", reference, current))
		// Measured: about 722 KB before and 224 KB after (-69%). Require at least -50% so the
		// test is a regression guard without being tied to the exact numbers.
		Expect(current).To(BeNumerically("<", reference/2))
	})

	It("reports how long a conversion takes", Label("measurement"), func() {
		pod := loadHeavyPod()
		experiment := gmeasure.NewExperiment("MapToTable on a 16 KB pod")
		AddReportEntry(experiment.Name, experiment)

		experiment.Sample(func(_ int) {
			experiment.MeasureDuration("reference", func() { _ = referenceMapToTable(pod) })
			experiment.MeasureDuration("current", func() { _ = lua_modules.MapToTable(pod) })
		}, gmeasure.SamplingConfig{N: 200})
	})
})

// BenchmarkMapToTableHeavyPod is kept as a plain Go benchmark so benchstat can compare runs:
// go test ./lib/lua -run '^$' -bench MapToTableHeavyPod -benchmem
func BenchmarkMapToTableHeavyPod(b *testing.B) {
	m, err := readHeavyPod()
	if err != nil {
		b.Fatal(err)
	}
	b.Run("reference", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = referenceMapToTable(m)
		}
	})
	b.Run("current", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = lua_modules.MapToTable(m)
		}
	})
}
