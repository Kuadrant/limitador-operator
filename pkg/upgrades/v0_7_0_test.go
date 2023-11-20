package upgrades

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

func TestUpgradeDeploymentTov070(t *testing.T) {
	var (
		limitadorName      = "test-limitador"
		limitadorNamespace = "default"
	)
	logger := log.Log.WithName("upgrades_test")
	baseCtx := context.Background()
	ctx := logr.NewContext(baseCtx, logger)

	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = limitadorv1alpha1.AddToScheme(s)
	_ = v1.AddToScheme(s)

	// Create dummy Limitador object
	limitadorObj := &limitadorv1alpha1.Limitador{
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitadorName,
			Namespace: limitadorNamespace,
		},
	}

	// Create a dummy Deployment object
	newDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitador.DeploymentName(limitadorObj),
			Namespace: limitadorNamespace,
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	// Old Deployment to simulate the upgrade
	oldDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReleaseV060DeploymentName(limitadorObj),
			Namespace: limitadorNamespace,
		},
	}

	// Objects to track in the fake client.
	objs := []client.Object{limitadorObj, newDeployment, oldDeployment}

	// Create a fake client to mock API calls.
	clBuilder := fake.NewClientBuilder()
	cl := clBuilder.WithScheme(s).WithObjects(objs...).Build()

	_, err := UpgradeDeploymentTov070(ctx, cl, limitadorObj, client.ObjectKeyFromObject(newDeployment))
	if err != nil {
		t.Fatalf("UpgradeDeploymentTov070 failed: %v", err)
	}

	// Check if the old deployment was deleted
	oldDepKey := types.NamespacedName{Name: ReleaseV060DeploymentName(limitadorObj), Namespace: limitadorNamespace}
	err = cl.Get(ctx, oldDepKey, &appsv1.Deployment{})
	if err == nil {
		t.Fatal("Old deployment was not deleted")
	}
}

func TestUpgradeDeploymentTov070_NewDeploymentNotFound(t *testing.T) {
	var (
		limitadorName      = "test-limitador"
		limitadorNamespace = "default"
	)
	logger := log.Log.WithName("upgrades_test")
	baseCtx := context.Background()
	ctx := logr.NewContext(baseCtx, logger)

	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = limitadorv1alpha1.AddToScheme(s)
	_ = v1.AddToScheme(s)

	// Create dummy Limitador object
	limitadorObj := &limitadorv1alpha1.Limitador{
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitadorName,
			Namespace: limitadorNamespace,
		},
	}

	// Old Deployment to simulate the upgrade
	oldDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReleaseV060DeploymentName(limitadorObj),
			Namespace: limitadorNamespace,
		},
	}

	newDeployment := &appsv1.Deployment{}

	// Objects to track in the fake client.
	objs := []client.Object{limitadorObj, oldDeployment}

	// Create a fake client to mock API calls.
	clBuilder := fake.NewClientBuilder()
	cl := clBuilder.WithScheme(s).WithObjects(objs...).Build()

	// Testa situation where the new deployment does not exist
	_, err := UpgradeDeploymentTov070(ctx, cl, limitadorObj, client.ObjectKeyFromObject(newDeployment))
	if err != nil {
		t.Fatalf("UpgradeDeploymentTov070 failed: %v", err)
	}

	// Check if the old deployment still exists
	oldDepKey := types.NamespacedName{Name: ReleaseV060DeploymentName(limitadorObj), Namespace: limitadorNamespace}
	err = cl.Get(ctx, oldDepKey, &appsv1.Deployment{})
	if err != nil {
		t.Fatalf("Old deployment should still exist: %v", err)
	}
}

func TestUpgradeConfigMapTov070(t *testing.T) {
	var (
		limitadorName      = "test-limitador"
		limitadorNamespace = "default"
	)
	logger := log.Log.WithName("upgrades_test")
	baseCtx := context.Background()
	ctx := logr.NewContext(baseCtx, logger)

	s := runtime.NewScheme()
	_ = v1.AddToScheme(s)
	_ = limitadorv1alpha1.AddToScheme(s)

	// Create dummy Limitador object
	limitadorObj := &limitadorv1alpha1.Limitador{
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitadorName,
			Namespace: limitadorNamespace,
		},
	}

	// Old ConfigMap to simulate the upgrade
	oldConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReleaseV060LimitsConfigMapName(limitadorObj),
			Namespace: limitadorNamespace,
		},
	}

	// Objects to track in the fake client.
	objs := []client.Object{limitadorObj, oldConfigMap}

	// Create a fake client to mock API calls.
	clBuilder := fake.NewClientBuilder()
	cl := clBuilder.WithScheme(s).WithObjects(objs...).Build()

	err := UpgradeConfigMapTov070(ctx, cl, limitadorObj)
	if err != nil {
		t.Fatalf("UpgradeConfigMapTov070 failed: %v", err)
	}

	// Check if the old config map was deleted
	oldCmKey := types.NamespacedName{Name: ReleaseV060LimitsConfigMapName(limitadorObj), Namespace: limitadorNamespace}
	err = cl.Get(ctx, oldCmKey, &v1.ConfigMap{})
	if !apierrors.IsNotFound(err) {
		t.Fatal("Old ConfigMap Found")
	}
}
