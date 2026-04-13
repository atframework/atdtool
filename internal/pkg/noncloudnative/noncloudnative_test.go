package noncloudnative

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func fixturePath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	all := append([]string{filepath.Dir(file), "testdata"}, parts...)
	return filepath.Join(all...)
}

func TestLoadConfigFindsNestedDeployFile(t *testing.T) {
	cfg, err := LoadConfig([]string{fixturePath("default")})
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, cfg) || !assert.NotNil(t, cfg.Deploy) {
		return
	}

	assert.Equal(t, uint64(1), cfg.Deploy.WorldID)
	assert.Equal(t, uint64(2), cfg.Deploy.ZoneId)
	if !assert.Len(t, cfg.Deploy.Instance, 1) {
		return
	}
	assert.Equal(t, "echo", cfg.Deploy.Instance[0].Name)
	assert.Equal(t, "11", cfg.Deploy.Instance[0].TypeId)
}

func TestLoadConfigLaterPathOverridesEarlier(t *testing.T) {
	cfg, err := LoadConfig([]string{fixturePath("default"), fixturePath("override")})
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, cfg) || !assert.NotNil(t, cfg.Deploy) {
		return
	}

	assert.Equal(t, uint64(9), cfg.Deploy.WorldID)
	assert.Equal(t, uint64(8), cfg.Deploy.ZoneId)
	if !assert.Len(t, cfg.Deploy.Instance, 1) {
		return
	}
	assert.Equal(t, "override", cfg.Deploy.Instance[0].Name)
	assert.Equal(t, "33", cfg.Deploy.Instance[0].TypeId)
}

func TestLoadConfigReturnsErrorWhenDeployMissing(t *testing.T) {
	_, err := LoadConfig([]string{t.TempDir()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration file(deploy.yaml) not found")
}

func TestConfigToRenderValues(t *testing.T) {
	cfg := &Config{}
	got, err := cfg.ToRenderValues("1.2.65.3")
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, uint64(1), got["world_id"])
	assert.Equal(t, uint64(2), got["zone_id"])
	assert.Equal(t, uint64(3), got["instance_id"])
	assert.Equal(t, "1.2.65.3", got["bus_addr"])
	assert.Equal(t, runtime.GOOS, got["atdtool_running_platform"])
	assert.NotContains(t, got, "deploy")
}
