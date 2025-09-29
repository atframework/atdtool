package snowflake

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockWorkerIdGenerator for testing
type MockWorkerIdGenerator struct {
	id  int64
	err error
}

func (m *MockWorkerIdGenerator) Id() (int64, error) {
	return m.id, m.err
}

func TestNewSnowFlake(t *testing.T) {
	testCase := []struct {
		name      string
		generator WorkerIdGenerator
	}{
		{"nil generation", nil},
		{"custom generation", &MockWorkerIdGenerator{id: 1}},
	}

	assert := assert.New(t)
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSnowFlake(tc.generator)
			assert.NotNil(sf, "Expected non-nil Snowflake instance")
		})
	}
}

func TestNextVal(t *testing.T) {
	testCase := []struct {
		name      string
		generator *MockWorkerIdGenerator
		expectErr bool
	}{
		{"successful generation", &MockWorkerIdGenerator{id: 1}, false},
		{"invalid worker id less than 0", &MockWorkerIdGenerator{id: -1}, true},
		{"invalid worker id greater than max", &MockWorkerIdGenerator{id: int64(-1^(-1<<workeridBits)) + 1}, true},
		{"worker id generator error", &MockWorkerIdGenerator{err: fmt.Errorf("generator error")}, true},
	}

	assert := assert.New(t)
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSnowFlake(tc.generator)
			id, err := sf.NextVal()
			if tc.expectErr {
				assert.NotNil(err, "Expected error, got nil")
				if tc.generator.err != nil {
					assert.Equal(tc.generator.err, err, "Expected error: %v, got: %v", tc.generator.err, err)
				}
			} else {
				assert.Nil(err, "Expected nil, got error: %v", err)
				assert.NotEqual(0, id, "Expected non-zero ID")
			}
		})
	}
}

func BenchmarkSnowflake_NextVal(b *testing.B) {
	sf := NewSnowFlake(nil)

	assert := assert.New(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sf.NextVal()
		assert.Nil(err, "NextVal() error = %v", err)
	}
}
