package limitador

import (
	"fmt"

	"k8s.io/utils/env"
)

const (
	LimitadorRepository = "quay.io/kuadrant/limitador"
)

var (
	defaultImage = fmt.Sprintf("%s:%s", LimitadorRepository, "latest")
)

func GetLimitadorImage() string {
	return env.GetString("RELATED_IMAGE_LIMITADOR", defaultImage)
}
