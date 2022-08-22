package limitador

import (
	"fmt"

	"github.com/kuadrant/limitador-operator/pkg/helpers"
)

var (
	defaultImageVersion = fmt.Sprintf("%s:%s", LimitadorRepository, "latest")
)

func GetLimitadorImageVersion() string {
	return helpers.FetchEnv("RELATED_IMAGE_LIMITADOR", defaultImageVersion)
}
