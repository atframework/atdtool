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
}
