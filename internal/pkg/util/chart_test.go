package util

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/atframework/atdtool/internal/pkg/noncloudnative"
)

func fixturePath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	all := append([]string{filepath.Dir(file), "testdata"}, parts...)
	return filepath.Join(all...)
}

func asMap(t *testing.T, in any) map[string]any {
	t.Helper()
	m, ok := in.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", in)
	}
	return m
}

func TestMergeChartValuesPrecedenceAndModules(t *testing.T) {
	chartPath := fixturePath("charts", "basic")
	valuesPaths := []string{fixturePath("values", "default"), fixturePath("values", "dev")}

	got, err := MergeChartValues(chartPath, valuesPaths, nil, nil)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "service-dev", got["shared"])
	assert.Equal(t, "chart", got["chart_only"])
	assert.Equal(t, "global-dev", got["global_only"])
	assert.Equal(t, "service-dev", got["service_only"])
	assert.NotContains(t, got, "disabled")

	// Gap A: chart default wins over global.yaml for the same key
	assert.Equal(t, "chart-value", got["chart_global_overlap"])

	cache := asMap(t, got["cache"])
	assert.Equal(t, true, cache["enabled"])
	assert.Equal(t, "service-cache-dev", cache["shared"])
	assert.Equal(t, "chart-cache", cache["chart_only"])
	assert.Equal(t, "chart-cache", cache["from_chart"])
	assert.Equal(t, "global-cache-dev", cache["from_global"])
	assert.Equal(t, "only-global-default", cache["only_global"])
	assert.Equal(t, "service-cache-dev", cache["from_service"])
	assert.Equal(t, "only-service-default", cache["only_service"])
	assert.Equal(t, "module-cache-dev", cache["from_module"])
	assert.Equal(t, "only-module-dev", cache["only_module"])

	// Gap E: chart default wins over global.yaml inside a module key
	assert.Equal(t, "chart-cache-value", cache["chart_global_overlap"])
}

func TestMergeChartValuesUsesTypeNameForServiceFile(t *testing.T) {
	got, err := MergeChartValues(
		fixturePath("charts", "alias-type"),
		[]string{fixturePath("values", "default")},
		nil,
		nil,
	)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "alias-type-default", got["shared"])
}

func TestMergeChartValuesUsesFuncNameForServiceFile(t *testing.T) {
	got, err := MergeChartValues(
		fixturePath("charts", "alias-func"),
		[]string{fixturePath("values", "default")},
		nil,
		nil,
	)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "alias-func-default", got["shared"])
}

func TestMergeChartValuesCommandLineOverridesAndCanDisableModule(t *testing.T) {
	chartPath := fixturePath("charts", "basic")
	valuesPaths := []string{fixturePath("values", "default"), fixturePath("values", "dev")}

	t.Run("command line has highest precedence", func(t *testing.T) {
		got, err := MergeChartValues(chartPath, valuesPaths, map[string]any{
			"shared": "cli",
			"cache": map[string]any{
				"from_module": "cli",
			},
		}, nil)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "cli", got["shared"])
		cache := asMap(t, got["cache"])
		assert.Equal(t, "cli", cache["from_module"])
		assert.Equal(t, "only-module-dev", cache["only_module"])
	})

	t.Run("explicit disable skips module injection", func(t *testing.T) {
		got, err := MergeChartValues(chartPath, valuesPaths, map[string]any{
			"cache": map[string]any{
				"enabled": false,
			},
		}, nil)
		if !assert.NoError(t, err) {
			return
		}

		cache := asMap(t, got["cache"])
		assert.Equal(t, false, cache["enabled"])
		assert.NotContains(t, cache, "from_module")
		assert.NotContains(t, cache, "only_module")
		assert.Equal(t, "service-cache-dev", cache["shared"])
	})
}

func TestMergeChartValuesWithNonCloudNativeValues(t *testing.T) {
	got, err := MergeChartValues(
		fixturePath("charts", "basic"),
		[]string{fixturePath("values", "default")},
		nil,
		&noncloudnative.RenderValue{
			BusAddr: "3.4.5.6",
			Config:  &noncloudnative.Config{},
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, uint64(3), got["world_id"])
	assert.Equal(t, uint64(4), got["zone_id"])
	assert.Equal(t, uint64(6), got["instance_id"])
	assert.Equal(t, "3.4.5.6", got["bus_addr"])
	assert.Equal(t, runtime.GOOS, got["atdtool_running_platform"])
}
