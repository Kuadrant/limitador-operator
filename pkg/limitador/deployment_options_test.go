package limitador

import (
	"testing"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentCommand(t *testing.T) {
	t.Run("when no rate limit headers set in the spec command line args does not include --rate-limit-headers", func(subT *testing.T) {
		limObj := &limitadorv1alpha1.Limitador{
			TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{Name: "somename", Namespace: "somenamespace"},
			Spec:       limitadorv1alpha1.LimitadorSpec{},
		}

		command := DeploymentCommand(limObj, DeploymentStorageOptions{})
		assert.DeepEqual(subT, command,
			[]string{
				"limitador-server",
				"/home/limitador/etc/limitador-config.yaml",
				"memory",
			})
	})

	t.Run("when rate limit headers set in the spec command line args includes --rate-limit-headers", func(subT *testing.T) {
		limObj := &limitadorv1alpha1.Limitador{
			TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{Name: "somename", Namespace: "somenamespace"},
			Spec: limitadorv1alpha1.LimitadorSpec{
				RateLimitHeaders: &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0],
			},
		}

		command := DeploymentCommand(limObj, DeploymentStorageOptions{})
		assert.DeepEqual(subT, command,
			[]string{
				"limitador-server",
				"--rate-limit-headers",
				"DRAFT_VERSION_03",
				"/home/limitador/etc/limitador-config.yaml",
				"memory",
			})
	})
}
