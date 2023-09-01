package limitador

import (
	"fmt"

	"k8s.io/utils/env"
)

var (
	defaultImageVersion = fmt.Sprintf("%s:%s", LimitadorRepository, "latest")
)

func GetLimitadorImageVersion() string {
	return env.GetString("RELATED_IMAGE_LIMITADOR", defaultImageVersion)
}
