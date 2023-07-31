package limitador

import (
	"testing"

	"gotest.tools/assert"
)

func TestLimitadorDefaultImage(t *testing.T) {
	assert.Equal(t, GetLimitadorImageVersion(), "quay.io/kuadrant/limitador:latest")
}
