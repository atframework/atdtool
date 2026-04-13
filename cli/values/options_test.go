package values

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionsMergeValues(t *testing.T) {
	t.Run("parse nested values arrays and strings", func(t *testing.T) {
		opts := &Options{Values: []string{
			"log_level=DEBUG",
			"vector.enabled=true",
			"vector.source[0].name=normal",
			"ports={7001,7002}",
			"message=hello world",
		}}

		got, err := opts.MergeValues()
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "DEBUG", got["log_level"])
		assert.Equal(t, "hello world", got["message"])
		assert.Equal(t, []interface{}{int64(7001), int64(7002)}, got["ports"])

		vector, ok := got["vector"].(map[string]interface{})
		if !assert.True(t, ok) {
			return
		}
		assert.Equal(t, true, vector["enabled"])

		source, ok := vector["source"].([]interface{})
		if !assert.True(t, ok) {
			return
		}
		first, ok := source[0].(map[string]interface{})
		if !assert.True(t, ok) {
			return
		}
		assert.Equal(t, "normal", first["name"])
	})

	t.Run("return error for invalid set syntax", func(t *testing.T) {
		opts := &Options{Values: []string{"invalid"}}
		_, err := opts.MergeValues()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed parsing --set data")
	})
}

func TestOptionsMergePaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	baseDir := t.TempDir()
	absoluteDir := filepath.Join(baseDir, "absolute")
	relativeDir := filepath.Join(baseDir, "relative")
	if err := os.MkdirAll(absoluteDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(relativeDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(baseDir); err != nil {
		t.Fatal(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	homeFixture, err := os.MkdirTemp(homeDir, "atdtool-mergepaths-")
	if err != nil {
		t.Skipf("skip home path expansion test: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(homeFixture)
	})

	homeRelative := strings.TrimPrefix(homeFixture, homeDir)
	homeRelative = strings.TrimPrefix(homeRelative, string(os.PathSeparator))
	tildePath := "~" + homeRelative

	opts := &Options{Paths: []string{absoluteDir, filepath.Base(relativeDir), tildePath}}
	got, err := opts.MergePaths()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []string{absoluteDir, relativeDir, homeFixture}, got)

	t.Run("return error when path missing", func(t *testing.T) {
		missing := &Options{Paths: []string{filepath.Join(baseDir, "not-found")}}
		paths, err := missing.MergePaths()
		assert.Error(t, err)
		assert.Empty(t, paths)
		assert.Contains(t, err.Error(), "is not exist")
	})

	t.Run("return all errors when multiple paths missing", func(t *testing.T) {
		missing := &Options{Paths: []string{
			filepath.Join(baseDir, "missing-a"),
			filepath.Join(baseDir, "missing-b"),
		}}
		paths, err := missing.MergePaths()
		assert.Error(t, err)
		assert.Empty(t, paths)
		assert.Contains(t, err.Error(), "missing-a")
		assert.Contains(t, err.Error(), "missing-b")
	})
}
