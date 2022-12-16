package limitador

import (
	"testing"

	"gotest.tools/assert"
)

func TestLimitadorDefaulImage(t *testing.T) {
	assert.Equal(t, GetLimitadorImageVersion(), "quay.io/kuadrant/limitador:v1.0.0")
}
