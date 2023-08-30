package controllers

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller", func() {
	const (
		LimitadorNamespace = "default"
		LimitadorReplicas  = 2
		LimitadorImage     = "quay.io/kuadrant/limitador"
		LimitadorVersion   = "0.3.0"
		LimitadorHTTPPort  = 8000
		LimitadorGRPCPort  = 8001

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	httpPortNumber := int32(LimitadorHTTPPort)
	grpcPortNumber := int32(LimitadorGRPCPort)

	replicas := LimitadorReplicas
	version := LimitadorVersion
	httpPort := &limitadorv1alpha1.TransportProtocol{Port: &httpPortNumber}
	grpcPort := &limitadorv1alpha1.TransportProtocol{Port: &grpcPortNumber}

	limits := []limitadorv1alpha1.RateLimit{
		{
			Conditions: []string{"req.method == GET"},
			MaxValue:   10,
			Namespace:  "test-namespace",
			Seconds:    60,
			Variables:  []string{"user_id"},
		},
		{
			Conditions: []string{"req.method == POST"},
			MaxValue:   5,
			Namespace:  "test-namespace",
			Seconds:    60,
			Variables:  []string{"user_id"},
		},
	}

	newLimitador := func() *limitadorv1alpha1.Limitador {
		// The name can't start with a number.
		name := "a" + string(uuid.NewUUID())

		return &limitadorv1alpha1.Limitador{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Limitador",
				APIVersion: "limitador.kuadrant.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: LimitadorNamespace,
			},
			Spec: limitadorv1alpha1.LimitadorSpec{
				Replicas: &replicas,
				Version:  &version,
				Listener: &limitadorv1alpha1.Listener{
					HTTP: httpPort,
					GRPC: grpcPort,
				},
				Limits: limits,
			},
		}
	}

	deletePropagationPolicy := client.PropagationPolicy(metav1.DeletePropagationForeground)

	Context("Creating a new empty Limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			limitadorObj.Spec = limitadorv1alpha1.LimitadorSpec{}
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should create a Limitador service with default ports", func() {
			createdLimitadorService := v1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.ServiceName(limitadorObj),
					},
					&createdLimitadorService)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(len(createdLimitadorService.Spec.Ports)).Should(Equal(2))
			Expect(createdLimitadorService.Spec.Ports[0].Name).Should(Equal("http"))
			Expect(createdLimitadorService.Spec.Ports[0].Port).Should(Equal(limitadorv1alpha1.DefaultServiceHTTPPort))
			Expect(createdLimitadorService.Spec.Ports[1].Name).Should(Equal("grpc"))
			Expect(createdLimitadorService.Spec.Ports[1].Port).Should(Equal(limitadorv1alpha1.DefaultServiceGRPCPort))
		})
	})

	Context("Creating a new Limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should create a new deployment with the right settings", func() {
			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitadorDeployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(*createdLimitadorDeployment.Spec.Replicas).Should(
				Equal((int32)(LimitadorReplicas)),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Image).Should(
				Equal(LimitadorImage + ":" + LimitadorVersion),
			)
			// It should contain at least the limits file
			Expect(len(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Command) > 1).Should(BeTrue())
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Command[1]).Should(
				Equal("/home/limitador/etc/limitador-config.yaml"),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath).Should(
				Equal("/home/limitador/etc/"),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).Should(
				Equal(limitador.LimitsCMNamePrefix + limitadorObj.Name),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Command).Should(
				// asserts request headers command line arg is not there
				Equal(
					[]string{
						"limitador-server",
						"/home/limitador/etc/limitador-config.yaml",
						"memory",
					},
				),
			)
		})

		It("Should create a Limitador service", func() {
			createdLimitadorService := v1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.ServiceName(limitadorObj),
					},
					&createdLimitadorService)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should build the correct Status", func() {
			createdLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() *limitadorv1alpha1.LimitadorService {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: limitadorObj.Namespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitador)
				if err != nil {
					return nil
				}
				return createdLimitador.Status.Service
			}, timeout, interval).Should(Equal(&limitadorv1alpha1.LimitadorService{
				Host: "limitador-" + limitadorObj.Name + ".default.svc.cluster.local",
				Ports: limitadorv1alpha1.Ports{
					GRPC: grpcPortNumber,
					HTTP: httpPortNumber,
				},
			}))

		})

		It("Should create a ConfigMap with the correct limits", func() {
			createdConfigMap := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.LimitsCMNamePrefix + limitadorObj.Name,
					},
					&createdConfigMap)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(createdConfigMap.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err == nil)
			Expect(cmLimits).To(Equal(limits))
		})
	})

	Context("Updating a limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should modify the limitador deployment", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			replicas = LimitadorReplicas + 1
			updatedLimitador.Spec.Replicas = &replicas
			version = "latest"
			updatedLimitador.Spec.Version = &version

			Expect(k8sClient.Update(context.TODO(), &updatedLimitador)).Should(Succeed())
			updatedLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitadorDeployment)

				if err != nil {
					return false
				}

				correctReplicas := *updatedLimitadorDeployment.Spec.Replicas == LimitadorReplicas+1
				correctImage := updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Image == LimitadorImage+":latest"

				return correctReplicas && correctImage
			}, timeout, interval).Should(BeTrue())
		})

		It("Should modify the ConfigMap accordingly", func() {
			originalCM := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.LimitsCMNamePrefix + limitadorObj.Name,
					},
					&originalCM)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			newLimits := []limitadorv1alpha1.RateLimit{
				{
					Conditions: []string{"req.method == GET"},
					MaxValue:   100,
					Namespace:  "test-namespace",
					Seconds:    60,
					Variables:  []string{"user_id"},
				},
			}
			updatedLimitador.Spec.Limits = newLimits
			Expect(k8sClient.Update(context.TODO(), &updatedLimitador)).Should(Succeed())

			updatedLimitadorConfigMap := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.LimitsCMNamePrefix + limitadorObj.Name,
					},
					&updatedLimitadorConfigMap)
				// wait until the CM has changed
				return err == nil && updatedLimitadorConfigMap.ResourceVersion != originalCM.ResourceVersion
			}, timeout, interval).Should(BeTrue())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(updatedLimitadorConfigMap.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err == nil)
			Expect(cmLimits).To(Equal(newLimits))
		})
	})

	Context("Creating a new Limitador object with rate limit headers", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			limitadorObj.Spec.RateLimitHeaders = &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0]
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should create a new deployment with rate limit headers command line arg", func() {
			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitadorDeployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			// It should contain at least the limits file
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Command).Should(
				// asserts request headers command line arg is not there
				Equal(
					[]string{
						"limitador-server",
						"--rate-limit-headers",
						"DRAFT_VERSION_03",
						"/home/limitador/etc/limitador-config.yaml",
						"memory",
					},
				),
			)
		})
	})

	Context("Reconciling command line args", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should modify the limitador deployment command line args", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(updatedLimitador.Spec.RateLimitHeaders).Should(BeNil())
			updatedLimitador.Spec.RateLimitHeaders = &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0]

			Expect(k8sClient.Update(context.TODO(), &updatedLimitador)).Should(Succeed())
			Eventually(func() bool {
				updatedLimitadorDeployment := appsv1.Deployment{}
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitadorDeployment)

				if err != nil {
					return false
				}

				return reflect.DeepEqual(updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Command,
					[]string{
						"limitador-server",
						"--rate-limit-headers",
						"DRAFT_VERSION_03",
						"/home/limitador/etc/limitador-config.yaml",
						"memory",
					})
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Modifying limitador deployment objects", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("User tries adding side-cars to deployment CR", func() {
			deploymentObj := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&deploymentObj)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(len(deploymentObj.Spec.Template.Spec.Containers)).To(Equal(1))
			containerObj := v1.Container{Name: LimitadorNamespace, Image: LimitadorNamespace}

			deploymentObj.Spec.Template.Spec.Containers = append(deploymentObj.Spec.Template.Spec.Containers, containerObj)

			Expect(k8sClient.Update(context.TODO(), &deploymentObj)).Should(Succeed())
			updateDeploymentObj := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updateDeploymentObj)

				return err == nil && len(updateDeploymentObj.Spec.Template.Spec.Containers) == 1
			}, timeout, interval).Should(BeTrue())

		})
	})
})
