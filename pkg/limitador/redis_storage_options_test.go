package limitador

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/log"
)

func helperGetSecretDataFromStringData(data map[string]string) map[string][]byte {
	result := map[string][]byte{}
	for k, v := range data {
		result[k] = []byte(v)
	}
	return result
}

func TestRedisDeploymentOptions(t *testing.T) {
	var (
		namespace = "some-ns"
	)

	logger := log.Log.WithName("redis_deployment_test")
	baseCtx := context.Background()
	ctx := logr.NewContext(baseCtx, logger)

	clientFactory := func(subT *testing.T, objs []client.Object) client.Client {
		s := scheme.Scheme
		err := appsv1.AddToScheme(s)
		assert.NilError(subT, err)

		// Create a fake client to mock API calls.
		clBuilder := fake.NewClientBuilder()
		return clBuilder.WithObjects(objs...).Build()
	}

	t.Run("redis secretRef missing", func(subT *testing.T) {
		cl := clientFactory(subT, nil)
		emptyRedisObj := limitadorv1alpha1.Redis{}
		_, err := RedisDeploymentOptions(ctx, cl, namespace, emptyRedisObj)
		assert.Error(subT, err, "there's no ConfigSecretRef set")
	})

	t.Run("redis secret resource missing", func(subT *testing.T) {
		cl := clientFactory(subT, nil)
		redisObj := limitadorv1alpha1.Redis{
			ConfigSecretRef: &v1.ObjectReference{Name: "notexisting", Namespace: namespace},
		}
		_, err := RedisDeploymentOptions(ctx, cl, namespace, redisObj)
		assert.Assert(subT, errors.IsNotFound(err))
	})

	t.Run("redis secret does not have URL field", func(subT *testing.T) {
		emptySecret := &v1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "redisSecret", Namespace: namespace},
			StringData: map[string]string{},
			Data:       map[string][]byte{},
			Type:       v1.SecretTypeOpaque,
		}
		cl := clientFactory(subT, []client.Object{emptySecret})
		redisObj := limitadorv1alpha1.Redis{
			ConfigSecretRef: &v1.ObjectReference{Name: "redisSecret", Namespace: namespace},
		}
		_, err := RedisDeploymentOptions(ctx, cl, namespace, redisObj)
		assert.Error(subT, err, "the storage config Secret doesn't have the `URL` field")
	})

	t.Run("basic redis options", func(subT *testing.T) {
		redisSecret := &v1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "redisSecret", Namespace: namespace},
			StringData: map[string]string{"URL": "redis://example.com:6379"},
			Type:       v1.SecretTypeOpaque,
		}
		redisSecret.Data = helperGetSecretDataFromStringData(redisSecret.StringData)

		cl := clientFactory(subT, []client.Object{redisSecret})
		redisObj := limitadorv1alpha1.Redis{
			ConfigSecretRef: &v1.ObjectReference{Name: "redisSecret", Namespace: namespace},
		}
		options, err := RedisDeploymentOptions(ctx, cl, namespace, redisObj)
		assert.NilError(subT, err)
		assert.DeepEqual(subT, options,
			DeploymentStorageOptions{
				Command: []string{"redis", "$(LIMITADOR_OPERATOR_REDIS_URL)"},
			},
		)
	})
}

func TestDeploymentEnvVar(t *testing.T) {
	type args struct {
		configSecretRef *v1.ObjectReference
	}
	tests := []struct {
		name    string
		args    args
		want    []v1.EnvVar
		wantErr bool
		error   string
	}{
		{
			name:    "Nil object passed",
			want:    nil,
			wantErr: true,
			error:   "there's no ConfigSecretRef set",
		},
		{
			name: "Receive correct Env settings",
			want: []v1.EnvVar{
				{
					Name: "LIMITADOR_OPERATOR_REDIS_URL",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test",
							},
							Key: "URL",
						},
					},
				},
			},
			wantErr: false,
			args: args{
				configSecretRef: &v1.ObjectReference{
					Name: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentEnvVar(tt.args.configSecretRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentEnvVar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && !strings.Contains(err.Error(), tt.error) {
				t.Errorf("DeploymentEnvVar() error = %v, expected error = %v", err, tt.error)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentEnvVar() got = %v, want %v", got, tt.want)
			}
		})
	}
}
