package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/atframework/atdtool/cli/values"
)

func fixturePath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	all := append([]string{filepath.Dir(file), "testdata"}, parts...)
	return filepath.Join(all...)
}

func TestTemplateOptionsRunRendersPerInstanceOutputs(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	o := &templateOptions{
		chartPath: fixturePath("charts"),
		outPath:   outDir,
		valOpts: values.Options{
			Paths: []string{fixturePath("values", "default")},
		},
	}

	err := o.run(stdout)
	if !assert.NoError(t, err) {
		return
	}

	assert.Contains(t, stdout.String(), "create('echo', '1.2.42.3') configuration success")
	assert.Contains(t, stdout.String(), "create('echo', '1.2.42.4') configuration success")

	cfg1 := filepath.Join(outDir, "echo", "cfg", "echo_1.2.42.3.yaml")
	cfg2 := filepath.Join(outDir, "echo", "cfg", "echo_1.2.42.4.yaml")
	script1 := filepath.Join(outDir, "echo", "bin", "start_1.2.42.3.sh")

	cfg1Data, err := os.ReadFile(cfg1)
	if !assert.NoError(t, err) {
		return
	}
	cfg2Data, err := os.ReadFile(cfg2)
	if !assert.NoError(t, err) {
		return
	}
	script1Data, err := os.ReadFile(script1)
	if !assert.NoError(t, err) {
		return
	}

	cfg1Text := string(cfg1Data)
	cfg2Text := string(cfg2Data)
	assert.Contains(t, cfg1Text, "type_id: 42")
	assert.Contains(t, cfg1Text, "world_id: 1")
	assert.Contains(t, cfg1Text, "zone_id: 2")
	assert.Contains(t, cfg1Text, "instance_id: 3")
	assert.Contains(t, cfg1Text, "bus_addr: 1.2.42.3")
	assert.Contains(t, cfg1Text, "shared: service")
	assert.Contains(t, cfg1Text, "service_only: service")
	assert.Contains(t, cfg1Text, "extra_enabled: true")
	assert.Contains(t, cfg1Text, "extra_from_module: module-default")
	assert.Contains(t, cfg2Text, "instance_id: 4")
	assert.Contains(t, cfg2Text, "bus_addr: 1.2.42.4")
	assert.Contains(t, string(script1Data), "1.2.42.3 1 2")
}

func TestTemplateOptionsRunSupportsGlobalOverrides(t *testing.T) {
	tests := []struct {
		name      string
		values    []string
		wantWorld string
		wantZone  string
		wantBus   string
	}{
		{
			name:      "int values",
			values:    []string{"global.world_id=9", "global.zone_id=10"},
			wantWorld: "9",
			wantZone:  "10",
			wantBus:   "9.10.42.3",
		},
		{
			name:      "string values",
			values:    []string{`global.world_id="11"`, `global.zone_id="12"`},
			wantWorld: "11",
			wantZone:  "12",
			wantBus:   "11.12.42.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := t.TempDir()
			o := &templateOptions{
				chartPath: fixturePath("charts"),
				outPath:   outDir,
				valOpts: values.Options{
					Paths:  []string{fixturePath("values", "default")},
					Values: tt.values,
				},
			}

			err := o.run(&bytes.Buffer{})
			if !assert.NoError(t, err) {
				return
			}

			cfgPath := filepath.Join(outDir, "echo", "cfg", fmt.Sprintf("echo_%s.yaml", tt.wantBus))
			cfgData, err := os.ReadFile(cfgPath)
			if !assert.NoError(t, err) {
				return
			}
			text := string(cfgData)

			assert.Contains(t, text, "world_id: "+tt.wantWorld)
			assert.Contains(t, text, "zone_id: "+tt.wantZone)
			assert.Contains(t, text, "bus_addr: "+tt.wantBus)
		})
	}
}

func TestConvertToUint64Opt(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      uint64
		wantError bool
	}{
		{name: "uint64", input: uint64(7), want: 7},
		{name: "int64", input: int64(8), want: 8},
		{name: "string", input: "9", want: 9},
		{name: "quoted string", input: `"10"`, want: 10},
		{name: "invalid", input: "abc", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToUint64Opt("world_id", tt.input)
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateOptionsRunRequiresOutputPath(t *testing.T) {
	o := &templateOptions{
		chartPath: fixturePath("charts"),
		valOpts: values.Options{
			Paths: []string{fixturePath("values", "default")},
		},
	}

	err := o.run(&bytes.Buffer{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outPath not found")
}
