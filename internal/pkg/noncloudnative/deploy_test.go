package noncloudnative

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBusAddr(t *testing.T) {
	t.Run("parse valid bus address", func(t *testing.T) {
		got, err := parseBusAddr("1.2.65.3")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, []uint64{1, 2, 65, 3}, got)
	})

	t.Run("reject invalid segment count", func(t *testing.T) {
		_, err := parseBusAddr("1.2.3")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "should be a.b.c.d")
	})

	t.Run("reject non numeric segment", func(t *testing.T) {
		_, err := parseBusAddr("1.2.x.4")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "can not convert x to uint64")
	})

	t.Run("all zeros", func(t *testing.T) {
		got, err := parseBusAddr("0.0.0.0")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, []uint64{0, 0, 0, 0}, got)
	})

	t.Run("large uint64 value", func(t *testing.T) {
		got, err := parseBusAddr("18446744073709551615.0.0.1")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, uint64(18446744073709551615), got[0])
	})

	t.Run("leading zeros are valid", func(t *testing.T) {
		got, err := parseBusAddr("001.002.003.004")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, []uint64{1, 2, 3, 4}, got)
	})

	t.Run("reject empty string", func(t *testing.T) {
		_, err := parseBusAddr("")
		assert.Error(t, err)
	})

	t.Run("reject too many segments", func(t *testing.T) {
		_, err := parseBusAddr("1.2.3.4.5")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "should be a.b.c.d")
	})

	t.Run("reject overflow", func(t *testing.T) {
		_, err := parseBusAddr("18446744073709551616.0.0.1")
		assert.Error(t, err)
	})
}
