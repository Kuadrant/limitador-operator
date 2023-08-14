package reconcilers_test

import (
	"github.com/kuadrant/limitador-operator/pkg/reconcilers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Deployment", func() {
	var desired *appsv1.Deployment

	BeforeEach(func() {
		desired = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "test",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "expected",
							},
						},
					},
				},
			},
		}
	})
	Describe("DeploymentContainerListMutator()", func() {
		It("Container image length is correct", func() {
			existing := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample",
					Namespace: "test",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "expected",
								},
							},
						},
					},
				},
			}

			result := reconcilers.DeploymentContainerListMutator(desired, existing)

			Expect(result).To(Equal(false))

		})

		It("Container spec has too many containers", func() {
			existing := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample",
					Namespace: "test",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "expected",
								},
								{
									Name: "unexpected",
								},
							},
						},
					},
				},
			}

			result := reconcilers.DeploymentContainerListMutator(desired, existing)

			Expect(result).To(Equal(true))
			Expect(len(existing.Spec.Template.Spec.Containers)).To(Equal(len(desired.Spec.Template.Spec.Containers)))

		})
	})
})
