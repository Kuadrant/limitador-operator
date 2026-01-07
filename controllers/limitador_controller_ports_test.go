package controllers

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages ports", func() {
	const (
		nodeTimeOut = NodeTimeout(time.Second * 30)
		specTimeOut = SpecTimeout(time.Minute * 2)
	)

	var testNamespace string

	BeforeEach(func(ctx SpecContext) {
		CreateNamespaceWithContext(ctx, &testNamespace)
	})

	AfterEach(func(ctx SpecContext) {
		DeleteNamespaceWithContext(ctx, &testNamespace)
	}, nodeTimeOut)

	Context("Creating a new Limitador object with specific ports", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		httpPortNumber := limitadorv1alpha1.DefaultServiceHTTPPort + 100
		grpcPortNumber := limitadorv1alpha1.DefaultServiceGRPCPort + 100

		httpPort := &limitadorv1alpha1.TransportProtocol{Port: &httpPortNumber}
		grpcPort := &limitadorv1alpha1.TransportProtocol{Port: &grpcPortNumber}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Listener = &limitadorv1alpha1.Listener{
				HTTP: httpPort, GRPC: grpcPort,
			}
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should configure k8s resources with the custom ports", func(ctx SpecContext) {
			// Deployment ports
			// Deployment command line
			// Deployment probes
			// Limitador CR status
			// Service

			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Ports).To(ContainElements(
				v1.ContainerPort{
					Name: "http", ContainerPort: httpPortNumber, Protocol: v1.ProtocolTCP,
				},
				v1.ContainerPort{
					Name: "grpc", ContainerPort: grpcPortNumber, Protocol: v1.ProtocolTCP,
				},
			))

			Expect(deployment.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"--http-port",
					strconv.Itoa(int(httpPortNumber)),
					"--rls-port",
					strconv.Itoa(int(grpcPortNumber)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				),
			)

			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.
				ProbeHandler.HTTPGet).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.
				ProbeHandler.HTTPGet.Port).To(Equal(intstr.FromInt(int(httpPortNumber))))

			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.
				ProbeHandler.HTTPGet).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.
				ProbeHandler.HTTPGet.Port).To(Equal(intstr.FromInt(int(httpPortNumber))))

			limitadorCR := &limitadorv1alpha1.Limitador{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(limitadorObj), limitadorCR)
			Expect(err).NotTo(HaveOccurred())
			Expect(limitadorCR.Status.Service).NotTo(BeNil())
			Expect(limitadorCR.Status.Service.Ports).To(Equal(
				limitadorv1alpha1.Ports{GRPC: grpcPortNumber, HTTP: httpPortNumber},
			))

			service := &v1.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: testNamespace,
				Name:      limitador.ServiceName(limitadorObj),
			}, service)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Spec.Ports).To(ContainElements(
				v1.ServicePort{
					Name: "http", Port: httpPortNumber, Protocol: v1.ProtocolTCP,
					TargetPort: intstr.FromString("http"),
				},
				v1.ServicePort{
					Name: "grpc", Port: grpcPortNumber, Protocol: v1.ProtocolTCP,
					TargetPort: intstr.FromString("grpc"),
				},
			))
		}, specTimeOut)
	})

	Context("Updating limitador object with new custom ports", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		httpPortNumber := limitadorv1alpha1.DefaultServiceHTTPPort + 100
		grpcPortNumber := limitadorv1alpha1.DefaultServiceGRPCPort + 100

		httpPort := &limitadorv1alpha1.TransportProtocol{Port: &httpPortNumber}
		grpcPort := &limitadorv1alpha1.TransportProtocol{Port: &grpcPortNumber}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify the k8s resources with the custom ports", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Ports).To(ContainElements(
				v1.ContainerPort{
					Name: "http", ContainerPort: limitadorv1alpha1.DefaultServiceHTTPPort, Protocol: v1.ProtocolTCP,
				},
				v1.ContainerPort{
					Name: "grpc", ContainerPort: limitadorv1alpha1.DefaultServiceGRPCPort, Protocol: v1.ProtocolTCP,
				},
			))

			Expect(deployment.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				),
			)

			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.
				ProbeHandler.HTTPGet).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.
				ProbeHandler.HTTPGet.Port).To(Equal(
				intstr.FromInt(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
			))

			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.
				ProbeHandler.HTTPGet).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.
				ProbeHandler.HTTPGet.Port).To(Equal(
				intstr.FromInt(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
			))

			limitadorCR := &limitadorv1alpha1.Limitador{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(limitadorObj), limitadorCR)
			Expect(err).NotTo(HaveOccurred())
			Expect(limitadorCR.Status.Service).NotTo(BeNil())
			Expect(limitadorCR.Status.Service.Ports).To(Equal(
				limitadorv1alpha1.Ports{
					GRPC: limitadorv1alpha1.DefaultServiceGRPCPort,
					HTTP: limitadorv1alpha1.DefaultServiceHTTPPort,
				},
			))

			service := &v1.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: testNamespace,
				Name:      limitador.ServiceName(limitadorObj),
			}, service)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Spec.Ports).To(ContainElements(
				v1.ServicePort{
					Name: "http", Port: limitadorv1alpha1.DefaultServiceHTTPPort,
					Protocol: v1.ProtocolTCP, TargetPort: intstr.FromString("http"),
				},
				v1.ServicePort{
					Name: "grpc", Port: limitadorv1alpha1.DefaultServiceGRPCPort,
					Protocol: v1.ProtocolTCP, TargetPort: intstr.FromString("grpc"),
				},
			))

			// Let's update limitador CR
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace, Name: limitadorObj.Name,
				}, &updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Listener = &limitadorv1alpha1.Listener{
					HTTP: httpPort, GRPC: grpcPort,
				}

				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newDeployment := appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &newDeployment)).To(Succeed())

				g.Expect(newDeployment.Spec.Template.Spec.Containers[0].Ports).To(ContainElements(v1.ContainerPort{
					Name: "http", ContainerPort: httpPortNumber, Protocol: v1.ProtocolTCP,
				}))

				g.Expect(newDeployment.Spec.Template.Spec.Containers[0].Ports).To(ContainElements(v1.ContainerPort{
					Name: "grpc", ContainerPort: grpcPortNumber, Protocol: v1.ProtocolTCP,
				}))

				g.Expect(newDeployment.Spec.Template.Spec.Containers[0].Args).To(Equal([]string{
					"--http-port",
					strconv.Itoa(int(httpPortNumber)),
					"--rls-port",
					strconv.Itoa(int(grpcPortNumber)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				}))

				g.Expect(newDeployment.Spec.Template.Spec.Containers[0].LivenessProbe.ProbeHandler.HTTPGet.Port).To(Equal(intstr.FromInt32(httpPortNumber)))
				g.Expect(newDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe.ProbeHandler.HTTPGet.Port).To(Equal(intstr.FromInt32(httpPortNumber)))
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newLimitador := &limitadorv1alpha1.Limitador{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(limitadorObj), newLimitador)).To(Succeed())
				g.Expect(newLimitador.Status.Service).NotTo(BeNil())
				g.Expect(newLimitador.Status.Service.Ports).To(Equal(limitadorv1alpha1.Ports{GRPC: grpcPortNumber, HTTP: httpPortNumber}))
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newService := &v1.Service{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.ServiceName(limitadorObj),
				}, newService)).To(Succeed())

				g.Expect(newService.Spec.Ports).To(ContainElements(v1.ServicePort{
					Name: "http", Port: httpPortNumber, Protocol: v1.ProtocolTCP,
					TargetPort: intstr.FromString("http"),
				}))

				g.Expect(newService.Spec.Ports).To(ContainElements(v1.ServicePort{
					Name: "grpc", Port: grpcPortNumber, Protocol: v1.ProtocolTCP,
					TargetPort: intstr.FromString("grpc"),
				}))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
