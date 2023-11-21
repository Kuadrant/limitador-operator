package controllers

import (
	"context"
	"reflect"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages ports", func() {

	var testNamespace string

	BeforeEach(func() {
		CreateNamespace(&testNamespace)
	})

	AfterEach(DeleteNamespaceCallback(&testNamespace))

	Context("Creating a new Limitador object with specific ports", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		var httpPortNumber int32 = limitadorv1alpha1.DefaultServiceHTTPPort + 100
		var grpcPortNumber int32 = limitadorv1alpha1.DefaultServiceGRPCPort + 100

		httpPort := &limitadorv1alpha1.TransportProtocol{Port: &httpPortNumber}
		grpcPort := &limitadorv1alpha1.TransportProtocol{Port: &grpcPortNumber}

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Listener = &limitadorv1alpha1.Listener{
				HTTP: httpPort, GRPC: grpcPort,
			}
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should configure k8s resources with the custom ports", func() {
			// Deployment ports
			// Deployment command line
			// Deployment probes
			// Limitador CR status
			// Service

			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Ports).To(ContainElements(
				v1.ContainerPort{
					Name: "http", ContainerPort: httpPortNumber, Protocol: v1.ProtocolTCP,
				},
				v1.ContainerPort{
					Name: "grpc", ContainerPort: grpcPortNumber, Protocol: v1.ProtocolTCP,
				},
			))

			Expect(deployment.Spec.Template.Spec.Containers[0].Command).To(
				HaveExactElements(
					"limitador-server",
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
			err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(limitadorObj), limitadorCR)
			Expect(err).NotTo(HaveOccurred())
			Expect(limitadorCR.Status.Service).NotTo(BeNil())
			Expect(limitadorCR.Status.Service.Ports).To(Equal(
				limitadorv1alpha1.Ports{GRPC: grpcPortNumber, HTTP: httpPortNumber},
			))

			service := &v1.Service{}
			err = k8sClient.Get(context.TODO(), types.NamespacedName{
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
		})
	})

	Context("Updating limitador object with new custom ports", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		var httpPortNumber int32 = limitadorv1alpha1.DefaultServiceHTTPPort + 100
		var grpcPortNumber int32 = limitadorv1alpha1.DefaultServiceGRPCPort + 100

		httpPort := &limitadorv1alpha1.TransportProtocol{Port: &httpPortNumber}
		grpcPort := &limitadorv1alpha1.TransportProtocol{Port: &grpcPortNumber}

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify the k8s resources with the custom ports", func() {
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Ports).To(ContainElements(
				v1.ContainerPort{
					Name: "http", ContainerPort: limitadorv1alpha1.DefaultServiceHTTPPort, Protocol: v1.ProtocolTCP,
				},
				v1.ContainerPort{
					Name: "grpc", ContainerPort: limitadorv1alpha1.DefaultServiceGRPCPort, Protocol: v1.ProtocolTCP,
				},
			))

			Expect(deployment.Spec.Template.Spec.Containers[0].Command).To(
				HaveExactElements(
					"limitador-server",
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
			err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(limitadorObj), limitadorCR)
			Expect(err).NotTo(HaveOccurred())
			Expect(limitadorCR.Status.Service).NotTo(BeNil())
			Expect(limitadorCR.Status.Service.Ports).To(Equal(
				limitadorv1alpha1.Ports{
					GRPC: limitadorv1alpha1.DefaultServiceGRPCPort,
					HTTP: limitadorv1alpha1.DefaultServiceHTTPPort,
				},
			))

			service := &v1.Service{}
			err = k8sClient.Get(context.TODO(), types.NamespacedName{
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
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace, Name: limitadorObj.Name,
				}, &updatedLimitador)

				if err != nil {
					return false
				}

				updatedLimitador.Spec.Listener = &limitadorv1alpha1.Listener{
					HTTP: httpPort, GRPC: grpcPort,
				}

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				newDeployment := appsv1.Deployment{}
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &newDeployment)

				if err != nil {
					return false
				}

				httpPortsMatch := slices.Index(newDeployment.Spec.Template.Spec.Containers[0].Ports,
					v1.ContainerPort{
						Name: "http", ContainerPort: httpPortNumber, Protocol: v1.ProtocolTCP,
					}) != -1

				grpcPortsMatch := slices.Index(newDeployment.Spec.Template.Spec.Containers[0].Ports,
					v1.ContainerPort{
						Name: "grpc", ContainerPort: grpcPortNumber, Protocol: v1.ProtocolTCP,
					}) != -1
				commandMatch := reflect.DeepEqual(newDeployment.Spec.Template.Spec.Containers[0].Command,
					[]string{
						"limitador-server",
						"--http-port",
						strconv.Itoa(int(httpPortNumber)),
						"--rls-port",
						strconv.Itoa(int(grpcPortNumber)),
						"/home/limitador/etc/limitador-config.yaml",
						"memory",
					})
				livenessProbeMatch := reflect.DeepEqual(newDeployment.Spec.Template.Spec.Containers[0].LivenessProbe.
					ProbeHandler.HTTPGet.Port, intstr.FromInt(int(httpPortNumber)))
				readinessProbeMatch := reflect.DeepEqual(newDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe.
					ProbeHandler.HTTPGet.Port, intstr.FromInt(int(httpPortNumber)))

				return !slices.Contains(
					[]bool{
						httpPortsMatch, grpcPortsMatch, commandMatch,
						livenessProbeMatch, readinessProbeMatch,
					}, false)
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				newLimitador := &limitadorv1alpha1.Limitador{}
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(limitadorObj), newLimitador)
				if err != nil {
					return false
				}

				if newLimitador.Status.Service == nil {
					return false
				}
				return reflect.DeepEqual(newLimitador.Status.Service.Ports,
					limitadorv1alpha1.Ports{GRPC: grpcPortNumber, HTTP: httpPortNumber},
				)
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				newService := &v1.Service{}
				err = k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.ServiceName(limitadorObj),
				}, newService)

				if err != nil {
					return false
				}

				httpPortsMatch := slices.Index(newService.Spec.Ports,
					v1.ServicePort{
						Name: "http", Port: httpPortNumber, Protocol: v1.ProtocolTCP,
						TargetPort: intstr.FromString("http"),
					}) != -1

				grpcPortsMatch := slices.Index(newService.Spec.Ports,
					v1.ServicePort{
						Name: "grpc", Port: grpcPortNumber, Protocol: v1.ProtocolTCP,
						TargetPort: intstr.FromString("grpc"),
					}) != -1

				return !slices.Contains([]bool{httpPortsMatch, grpcPortsMatch}, false)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
