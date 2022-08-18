package limitador

import (
	"testing"

	"gotest.tools/assert"
)

func TestLimitadorDefaulImage(t *testing.T) {
	assert.Equal(t, GetLimitadorImageVersion(), "quay.io/3scale/limitador:latest")
}
