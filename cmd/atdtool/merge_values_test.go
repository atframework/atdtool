package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/atframework/atdtool/cli/values"
)

func TestMergeValuesOptionsRunOutputsYAML(t *testing.T) {
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "merged.yaml")

	o := &mergeValuesOptions{
		chartPath: fixturePath("charts", "echo"),
		outPath:   outFile,
		valOpts: values.Options{
			Paths: []string{fixturePath("values", "default")},
		},
	}

	err := o.run(&bytes.Buffer{})
	if !assert.NoError(t, err) {
		return
	}

	data, err := os.ReadFile(outFile)
	if !assert.NoError(t, err) {
		return
	}
	text := string(data)

	assert.Contains(t, text, "shared: service")
	assert.Contains(t, text, "service_only: service")
	assert.Contains(t, text, "type_name: echo")
}

func TestMergeValuesOptionsRunDefaultOutputPath(t *testing.T) {
	// When outPath is empty, output goes to <chartPath>/values.yaml.
	// Use a copy of the chart in a temp dir to avoid modifying testdata.
	tmpDir := t.TempDir()
	chartDir := filepath.Join(tmpDir, "echo")
	if err := copyDir(fixturePath("charts", "echo"), chartDir); err != nil {
		t.Fatalf("setup: %v", err)
	}

	o := &mergeValuesOptions{
		chartPath: chartDir,
		valOpts: values.Options{
			Paths: []string{fixturePath("values", "default")},
		},
	}

	err := o.run(&bytes.Buffer{})
	if !assert.NoError(t, err) {
		return
	}

	defaultOut := filepath.Join(chartDir, "values.yaml")
	data, err := os.ReadFile(defaultOut)
	if !assert.NoError(t, err) {
		return
	}

	assert.Contains(t, string(data), "shared: service")
}

func TestMergeValuesOptionsRunWithSet(t *testing.T) {
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "merged.yaml")

	o := &mergeValuesOptions{
		chartPath: fixturePath("charts", "echo"),
		outPath:   outFile,
		valOpts: values.Options{
			Paths:  []string{fixturePath("values", "default")},
			Values: []string{"shared=cli-override"},
		},
	}

	err := o.run(&bytes.Buffer{})
	if !assert.NoError(t, err) {
		return
	}

	data, err := os.ReadFile(outFile)
	if !assert.NoError(t, err) {
		return
	}

	assert.Contains(t, string(data), "shared: cli-override")
}

func TestMergeValuesOptionsRunOutputDirNoExtension(t *testing.T) {
	outDir := t.TempDir()

	o := &mergeValuesOptions{
		chartPath: fixturePath("charts", "echo"),
		outPath:   outDir,
		valOpts: values.Options{
			Paths: []string{fixturePath("values", "default")},
		},
	}

	err := o.run(&bytes.Buffer{})
	if !assert.NoError(t, err) {
		return
	}

	defaultOut := filepath.Join(outDir, "values.yaml")
	_, err = os.Stat(defaultOut)
	assert.NoError(t, err)
}

// copyDir copies all files from src to dst (non-recursive, sufficient for flat chart dirs).
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			if err := copyDir(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}
