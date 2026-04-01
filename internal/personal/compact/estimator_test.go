package compact

import (
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/assert"
)

func TestEstimateTextTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		text   string
		expect int
	}{
		{"hello world", 3},
		{"", 0},
		{"Hello, World! This is a test.", 9},
		{"A much longer string that represents typical code", 12},
	}
	for _, tt := range tests {
		result := textTokens(tt.text)
		assert.GreaterOrEqual(t, result, tt.expect-2)
		assert.LessOrEqual(t, result, tt.expect+2)
	}
}

func TestEstimateTokens_Empty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, EstimateTokens(nil))
	assert.Equal(t, 0, EstimateTokens([]fantasy.Message{}))
}

func TestFormatTokens(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "500", FormatTokens(500))
	assert.Equal(t, "1.5K", FormatTokens(1500))
	assert.Equal(t, "200.0K", FormatTokens(200000))
}
