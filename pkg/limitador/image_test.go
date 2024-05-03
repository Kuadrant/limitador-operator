package limitador

import (
	"testing"

	"gotest.tools/assert"
)

func TestLimitadorDefaultImage(t *testing.T) {
	assert.Equal(t, GetLimitadorImage(), "quay.io/kuadrant/limitador:latest")
}
