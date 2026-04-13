package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenGUIDRunDefault(t *testing.T) {
	var buf bytes.Buffer
	o := &genGUIDOptions{}
	err := o.run(&buf)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestGenGUIDRunSnowflake(t *testing.T) {
	var buf bytes.Buffer
	o := &genGUIDOptions{algorithm: "snowflake"}
	err := o.run(&buf)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestGenGUIDRunUnsupportedAlgorithm(t *testing.T) {
	var buf bytes.Buffer
	o := &genGUIDOptions{algorithm: "uuid"}
	err := o.run(&buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}
