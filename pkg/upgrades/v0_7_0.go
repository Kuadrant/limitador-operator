package upgrades

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/helpers"

	"github.com/go-logr/logr"
)

func UpgradeDeploymentTov070(ctx context.Context, cli client.Client, limitadorObj *limitadorv1alpha1.Limitador, newDeploymentKey client.ObjectKey) (ctrl.Result, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(1).Info("Upgrading Deployment to v0.7.0", "deployment", newDeploymentKey.Name)

	newDeployment := &appsv1.Deployment{}
	if err := cli.Get(ctx, newDeploymentKey, newDeployment); err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("New deployment not found")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	availableCondition := helpers.FindDeploymentStatusCondition(newDeployment.Status.Conditions, "Available")

	if availableCondition == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	if availableCondition.Status == v1.ConditionTrue {
		oldDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ReleaseV060DeploymentName(limitadorObj),
				Namespace: limitadorObj.Namespace,
			},
		}
		if err := cli.Delete(ctx, oldDeployment); err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func UpgradeConfigMapTov070(ctx context.Context, cli client.Client, limitadorObj *limitadorv1alpha1.Limitador) error {
	oldConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReleaseV060LimitsConfigMapName(limitadorObj),
			Namespace: limitadorObj.Namespace,
		},
	}

	if err := cli.Delete(ctx, oldConfigMap); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func ReleaseV060DeploymentName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return limitadorObj.Name
}

func ReleaseV060LimitsConfigMapName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("limits-config-%s", limitadorObj.Name)
}
