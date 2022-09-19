package helpers

import (
	"gotest.tools/assert"
	"testing"
)

func TestToKebabCase(t *testing.T) {
	assert.Equal(t, ToKebabCase("Ttl"), "ttl")
	assert.Equal(t, ToKebabCase("Ratio"), "ratio")
	assert.Equal(t, ToKebabCase("FlushPeriod"), "flush-period")
	assert.Equal(t, ToKebabCase("MaxCached"), "max-cached")
}
