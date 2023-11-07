package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

const (
	LimitadorNamespace = "default"
	timeout            = time.Second * 10
	interval           = time.Millisecond * 250
)

var _ = Describe("Limitador controller", func() {
	const (
		LimitadorReplicas              = 2
		LimitadorImage                 = "quay.io/kuadrant/limitador"
		LimitadorVersion               = "0.3.0"
		LimitadorHTTPPort              = 8000
		LimitadorGRPCPort              = 8001
		LimitadorMaxUnavailable        = 1
		LimitdaorUpdatedMaxUnavailable = 3
	)

	httpPortNumber := int32(LimitadorHTTPPort)
	grpcPortNumber := int32(LimitadorGRPCPort)

	maxUnavailable := &intstr.IntOrString{
		Type:   0,
		IntVal: LimitadorMaxUnavailable,
	}
	updatedMaxUnavailable := &intstr.IntOrString{
		Type:   0,
		IntVal: LimitdaorUpdatedMaxUnavailable,
	}

	replicas := LimitadorReplicas
	version := LimitadorVersion
	httpPort := &limitadorv1alpha1.TransportProtocol{Port: &httpPortNumber}
	grpcPort := &limitadorv1alpha1.TransportProtocol{Port: &grpcPortNumber}
	affinity := &v1.Affinity{
		PodAntiAffinity: &v1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: v1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pod": "label",
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	limits := []limitadorv1alpha1.RateLimit{
		{
			Conditions: []string{"req.method == 'GET'"},
			MaxValue:   10,
			Namespace:  "test-namespace",
			Seconds:    60,
			Variables:  []string{"user_id"},
			Name:       "useless",
		},
		{
			Conditions: []string{"req.method == 'POST'"},
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
				Affinity: affinity,
				Listener: &limitadorv1alpha1.Listener{
					HTTP: httpPort,
					GRPC: grpcPort,
				},
				Limits: limits,
				PodDisruptionBudget: &limitadorv1alpha1.PodDisruptionBudgetType{
					MaxUnavailable: maxUnavailable,
				},
			},
		}
	}

	deletePropagationPolicy := client.PropagationPolicy(metav1.DeletePropagationForeground)

	Context("Creating a new empty Limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			limitadorObj.Spec = limitadorv1alpha1.LimitadorSpec{}

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))
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
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))
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
				Equal("/home/limitador/etc"),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).Should(
				Equal(limitador.LimitsConfigMapName(limitadorObj)),
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
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Resources).Should(
				Equal(*limitadorObj.GetResourceRequirements()))
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Affinity).Should(
				Equal(affinity),
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
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					},
					&createdConfigMap)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(createdConfigMap.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err == nil)
			Expect(cmLimits).To(Equal(limits))
		})

		It("Should create a PodDisruptionBudget", func() {
			createdPdb := policyv1.PodDisruptionBudget{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.PodDisruptionBudgetName(limitadorObj),
					},
					&createdPdb)

				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdPdb.Spec.MaxUnavailable).To(Equal(maxUnavailable))
			Expect(createdPdb.Spec.Selector.MatchLabels).To(Equal(limitador.Labels(limitadorObj)))
		})
	})

	Context("Updating a limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))
		})

		It("Should modify the limitador deployment", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			replicas = LimitadorReplicas + 1
			version = "latest"
			resourceRequirements := &v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("30Mi"),
				},
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("400m"),
					v1.ResourceMemory: resource.MustParse("60Mi"),
				},
			}

			// Sometimes there can be a conflict due to stale resource if controller is still reconciling resource
			// from create event
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)
				if err != nil {
					return false
				}

				updatedLimitador.Spec.Replicas = &replicas
				updatedLimitador.Spec.Version = &version
				updatedLimitador.Spec.ResourceRequirements = resourceRequirements
				affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight = 99
				updatedLimitador.Spec.Affinity = affinity

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

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
				correctResources := reflect.DeepEqual(updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Resources, *resourceRequirements)
				correctAffinity := updatedLimitadorDeployment.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight == 99

				return correctReplicas && correctImage && correctResources && correctAffinity
			}, timeout, interval).Should(BeTrue())
		})

		It("Should modify limitador deployments if nil object set", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				if err != nil {
					return false
				}
				updatedLimitador.Spec.Affinity = nil

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil

			}, timeout, interval).Should(BeTrue())

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

				correctAffinity := updatedLimitadorDeployment.Spec.Template.Spec.Affinity == nil

				return correctAffinity
			}, timeout, interval).Should(BeTrue())
		})

		It("Should modify the ConfigMap accordingly", func() {
			originalCM := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					},
					&originalCM)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			updatedLimitador := limitadorv1alpha1.Limitador{}
			newLimits := []limitadorv1alpha1.RateLimit{
				{
					Conditions: []string{"req.method == GET"},
					MaxValue:   100,
					Namespace:  "test-namespace",
					Seconds:    60,
					Variables:  []string{"user_id"},
				},
			}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				updatedLimitador.Spec.Limits = newLimits

				if err != nil {
					return false
				}

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			updatedLimitadorConfigMap := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
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

		It("Updates the PodDisruptionBudget accordingly", func() {
			originalPdb := policyv1.PodDisruptionBudget{}

			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.PodDisruptionBudgetName(limitadorObj),
					},
					&originalPdb)

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

				if err != nil {
					return false
				}
				updatedLimitador.Spec.PodDisruptionBudget.MaxUnavailable = updatedMaxUnavailable

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			updatedPdb := policyv1.PodDisruptionBudget{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.PodDisruptionBudgetName(limitadorObj),
					},
					&updatedPdb)

				return err == nil && updatedPdb.ResourceVersion != originalPdb.ResourceVersion
			}, timeout, interval).Should(BeTrue())

			Expect(updatedPdb.Spec.MaxUnavailable).To(Equal(updatedMaxUnavailable))
		})
	})

	Context("Creating a new Limitador object with rate limit headers", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			limitadorObj.Spec.RateLimitHeaders = &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0]

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))
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

	Context("Reconciling command line args for rate limit headers", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))
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

				if err != nil {
					return false
				}

				if updatedLimitador.Spec.RateLimitHeaders != nil {
					return false
				}
				updatedLimitador.Spec.RateLimitHeaders = &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0]
				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

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

	Context("Reconciling command line args for telemetry", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))
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

				if err != nil {
					return false
				}

				if updatedLimitador.Spec.Telemetry != nil {
					return false
				}
				telemetry := limitadorv1alpha1.Telemetry("exhaustive")
				updatedLimitador.Spec.Telemetry = &telemetry
				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

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
						"--limit-name-in-labels",
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
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))
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

	// This test requires actual k8s cluster
	// It's testing implementation based on CRD x-kubernetes-validations extentions
	// used to validate custom resources using Common Expression Language (CEL)
	// https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules
	Context("Disk storage does not allow multiple replicas", func() {
		AfterEach(func() {
			limitadorObj := limitadorWithInvalidDiskReplicas()
			err := k8sClient.Delete(context.TODO(), limitadorObj)
			Expect(err == nil || errors.IsNotFound(err))
			Eventually(func() bool {
				var l limitadorv1alpha1.Limitador
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(limitadorObj), &l)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("resource is rejected", func() {
			limitadorObj := limitadorWithInvalidDiskReplicas()
			err := k8sClient.Create(context.TODO(), limitadorObj)
			Expect(err).To(HaveOccurred())
			Expect(errors.IsInvalid(err)).To(BeTrue())
		})
	})

	Context("Deploying limitador object with redis storage", func() {
		redisSecret := &v1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "redis", Namespace: LimitadorNamespace},
			StringData: map[string]string{"URL": "redis://example.com:6379"},
			Type:       v1.SecretTypeOpaque,
		}

		BeforeEach(func() {
			deployRedis()

			err := k8sClient.Create(context.Background(), redisSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				secret := &v1.Secret{}
				err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(redisSecret), secret)
				if err != nil {
					if errors.IsNotFound(err) {
						fmt.Fprintln(GinkgoWriter, "==== redis secret not found")
					} else {
						fmt.Fprintln(GinkgoWriter, "==== cannot read redis secret", "error", err)
					}

					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())
		})

		AfterEach(func() {
			unDeployRedis()
			limitadorObj := limitadorWithRedisStorage(client.ObjectKeyFromObject(redisSecret))
			err := k8sClient.Delete(context.TODO(), limitadorObj)
			Expect(err == nil || errors.IsNotFound(err))
			Eventually(func() bool {
				var l limitadorv1alpha1.Limitador
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(limitadorObj), &l)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			err = k8sClient.Delete(context.TODO(), redisSecret)
			Expect(err == nil || errors.IsNotFound(err))
			Eventually(func() bool {
				var s v1.Secret
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(redisSecret), &s)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("command line is correct", func() {
			limitadorObj := limitadorWithRedisStorage(client.ObjectKeyFromObject(redisSecret))
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())

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
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Command).To(
				Equal(
					[]string{
						"limitador-server",
						"/home/limitador/etc/limitador-config.yaml",
						"redis",
						"$(LIMITADOR_OPERATOR_REDIS_URL)",
					},
				),
			)
		})
	})

	Context("Deploying limitador object with redis cached storage", func() {
		redisSecret := &v1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: "redis", Namespace: LimitadorNamespace},
			StringData: map[string]string{"URL": "redis://example.com:6379"},
			Type:       v1.SecretTypeOpaque,
		}

		BeforeEach(func() {
			deployRedis()

			err := k8sClient.Create(context.Background(), redisSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				secret := &v1.Secret{}
				err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(redisSecret), secret)
				if err != nil {
					if errors.IsNotFound(err) {
						fmt.Fprintln(GinkgoWriter, "redis secret not found")
					} else {
						fmt.Fprintln(GinkgoWriter, "cannot read redis secret", "error", err)
					}

					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())
		})

		AfterEach(func() {
			unDeployRedis()
			limitadorObj := limitadorWithRedisCachedStorage(client.ObjectKeyFromObject(redisSecret))
			err := k8sClient.Delete(context.TODO(), limitadorObj)
			Expect(err == nil || errors.IsNotFound(err))
			Eventually(func() bool {
				var l limitadorv1alpha1.Limitador
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(limitadorObj), &l)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			err = k8sClient.Delete(context.TODO(), redisSecret)
			Expect(err == nil || errors.IsNotFound(err))
			Eventually(func() bool {
				var s v1.Secret
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(redisSecret), &s)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

		})

		It("command line is correct", func() {
			limitadorObj := limitadorWithRedisCachedStorage(client.ObjectKeyFromObject(redisSecret))
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())

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
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Command).To(
				Equal(
					[]string{
						"limitador-server",
						"/home/limitador/etc/limitador-config.yaml",
						"redis_cached",
						"$(LIMITADOR_OPERATOR_REDIS_URL)",
						"--ttl", "1",
						"--ratio", "2",
						"--flush-period", "3",
						"--max-cached", "4",
					},
				),
			)
		})
	})

	Context("Deploying limitador object with disk storage", func() {
		AfterEach(func() {
			limitadorObj := limitadorWithDiskStorage()
			err := k8sClient.Delete(context.TODO(), limitadorObj)
			Expect(err == nil || errors.IsNotFound(err))
			Eventually(func() bool {
				var l limitadorv1alpha1.Limitador
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(limitadorObj), &l)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("deployment is correct", func() {
			limitadorObj := limitadorWithDiskStorage()
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())

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

			Expect(len(deploymentObj.Spec.Template.Spec.Volumes)).To(Equal(2))
			Expect(deploymentObj.Spec.Template.Spec.Volumes[1]).To(
				Equal(
					v1.Volume{
						Name: limitador.DiskVolumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: limitador.PVCName(limitadorObj),
								ReadOnly:  false,
							},
						},
					},
				))

			Expect(len(deploymentObj.Spec.Template.Spec.Containers)).To(Equal(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Command).To(
				Equal(
					[]string{
						"limitador-server",
						"/home/limitador/etc/limitador-config.yaml",
						"disk",
						limitador.DiskPath,
					},
				),
			)
			Expect(len(deploymentObj.Spec.Template.Spec.Containers[0].VolumeMounts)).To(Equal(2))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].VolumeMounts[1]).To(
				Equal(
					v1.VolumeMount{
						ReadOnly:  false,
						Name:      limitador.DiskVolumeName,
						MountPath: limitador.DiskPath,
					},
				),
			)
		})

		It("pvc is correct", func() {
			limitadorObj := limitadorWithDiskStorage()
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())

			pvc := &v1.PersistentVolumeClaim{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      limitador.PVCName(limitadorObj),
						Namespace: LimitadorNamespace,
					},
					pvc)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(len(pvc.GetOwnerReferences())).To(Equal(1))
		})
	})
})

func limitadorWithRedisStorage(redisKey client.ObjectKey) *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-redis-storage", Namespace: LimitadorNamespace},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Storage: &limitadorv1alpha1.Storage{
				Redis: &limitadorv1alpha1.Redis{
					ConfigSecretRef: &v1.ObjectReference{
						Name:      redisKey.Name,
						Namespace: redisKey.Namespace,
					},
				},
			},
		},
	}
}

func limitadorWithRedisCachedStorage(key client.ObjectKey) *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-redis-cached-storage", Namespace: LimitadorNamespace},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Storage: &limitadorv1alpha1.Storage{
				RedisCached: &limitadorv1alpha1.RedisCached{
					ConfigSecretRef: &v1.ObjectReference{
						Name:      key.Name,
						Namespace: key.Namespace,
					},
					Options: &limitadorv1alpha1.RedisCachedOptions{
						TTL:         &[]int{1}[0],
						Ratio:       &[]int{2}[0],
						FlushPeriod: &[]int{3}[0],
						MaxCached:   &[]int{4}[0],
					},
				},
			},
		},
	}
}

func limitadorWithDiskStorage() *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-disk-storage", Namespace: LimitadorNamespace},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Storage: &limitadorv1alpha1.Storage{
				Disk: &limitadorv1alpha1.DiskSpec{},
			},
		},
	}
}

func limitadorWithInvalidDiskReplicas() *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-invalid-disk-replicas", Namespace: LimitadorNamespace},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Replicas: &[]int{2}[0],
			Storage: &limitadorv1alpha1.Storage{
				Disk: &limitadorv1alpha1.DiskSpec{},
			},
		},
	}
}

func deployRedis() {
	deployment := redisDeployment()
	Expect(k8sClient.Create(context.TODO(), deployment)).Should(Succeed())
	Eventually(func() bool {
		d := &appsv1.Deployment{}
		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(deployment), d)
		return err == nil
	}, timeout, interval).Should(BeTrue())

	service := redisService()
	Expect(k8sClient.Create(context.TODO(), service)).Should(Succeed())
	Eventually(func() bool {
		s := &v1.Service{}
		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(service), s)
		return err == nil
	}, timeout, interval).Should(BeTrue())
}

func unDeployRedis() {
	deployment := redisDeployment()
	err := k8sClient.Delete(context.TODO(), deployment)
	Expect(err == nil || errors.IsNotFound(err))
	Eventually(func() bool {
		var d appsv1.Deployment
		err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(deployment), &d)
		return errors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())

	service := redisService()
	err = k8sClient.Delete(context.TODO(), service)
	Expect(err == nil || errors.IsNotFound(err))
	Eventually(func() bool {
		s := &v1.Service{}
		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(service), s)
		return errors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

func redisDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: LimitadorNamespace,
			Labels:    map[string]string{"app": "redis"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "redis"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "redis"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Name: "redis", Image: "redis"}},
				},
			},
		},
	}
}

func redisService() *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: LimitadorNamespace,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"app": "redis"},
			Ports: []v1.ServicePort{
				{
					Name:       "redis",
					Protocol:   v1.ProtocolTCP,
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
			},
		},
	}
}
