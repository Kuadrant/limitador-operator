package controllers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var (
	ExpectedDefaultImage = fmt.Sprintf("%s:%s", limitador.LimitadorRepository, "latest")
)

var _ = Describe("Limitador controller", func() {
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

	Context("Creating a new basic limitador CR", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should create a Limitador service with default ports", func(ctx SpecContext) {
			createdLimitadorService := corev1.Service{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.ServiceName(limitadorObj),
					},
					&createdLimitadorService)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())
			Expect(createdLimitadorService.Spec.Ports).To(HaveLen(2))
			Expect(createdLimitadorService.Spec.Ports[0].Name).Should(Equal("http"))
			Expect(createdLimitadorService.Spec.Ports[0].Port).Should(Equal(limitadorv1alpha1.DefaultServiceHTTPPort))
			Expect(createdLimitadorService.Spec.Ports[1].Name).Should(Equal("grpc"))
			Expect(createdLimitadorService.Spec.Ports[1].Port).Should(Equal(limitadorv1alpha1.DefaultServiceGRPCPort))
		}, specTimeOut)

		It("Should create a new deployment with default settings", func(ctx SpecContext) {
			var expectedDefaultReplicas int32 = 1
			expectedDefaultResourceRequirements := corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			}

			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&createdLimitadorDeployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(*createdLimitadorDeployment.Spec.Replicas).Should(Equal(expectedDefaultReplicas))
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Image).Should(
				Equal(ExpectedDefaultImage),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath).Should(
				Equal("/home/limitador/etc"),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).Should(
				Equal(limitador.LimitsConfigMapName(limitadorObj)),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Args).Should(
				// asserts no additional command line arg is added
				HaveExactElements(
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Resources).To(
				Equal(expectedDefaultResourceRequirements))
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Affinity).To(BeNil())
		}, specTimeOut)

		It("Should build the correct Status", func(ctx SpecContext) {
			createdLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) *limitadorv1alpha1.LimitadorService {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: limitadorObj.Namespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitador)).To(Succeed())
				return createdLimitador.Status.Service
			}).WithContext(ctx).Should(Equal(&limitadorv1alpha1.LimitadorService{
				Host: fmt.Sprintf("%s.%s.svc.cluster.local", limitador.ServiceName(limitadorObj), testNamespace),
				Ports: limitadorv1alpha1.Ports{
					GRPC: limitadorv1alpha1.DefaultServiceGRPCPort,
					HTTP: limitadorv1alpha1.DefaultServiceHTTPPort,
				},
			}))
		}, specTimeOut)

		It("Should create a ConfigMap with empty limits", func(ctx SpecContext) {
			createdConfigMap := corev1.ConfigMap{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					},
					&createdConfigMap)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(createdConfigMap.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err).ToNot(HaveOccurred())
			Expect(cmLimits).To(BeEmpty())
		}, specTimeOut)

		It("Should have not created PodDisruptionBudget", func(ctx SpecContext) {
			pdb := &policyv1.PodDisruptionBudget{}
			err := k8sClient.Get(ctx,
				types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.PodDisruptionBudgetName(limitadorObj),
				}, pdb)
			// returns false when err is nil
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		}, specTimeOut)
	})

	Context("Creating a new Limitador object with rate limit headers", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.RateLimitHeaders = ptr.To(limitadorv1alpha1.RateLimitHeadersType("DRAFT_VERSION_03"))

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should create a new deployment with rate limit headers command line arg", func(ctx SpecContext) {
			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&createdLimitadorDeployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"--rate-limit-headers",
					"DRAFT_VERSION_03",
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				),
			)
		}, specTimeOut)
	})

	Context("Reconciling command line args for rate limit headers", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify the limitador deployment command line args", func(ctx SpecContext) {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)).To(Succeed())

				g.Expect(updatedLimitador.Spec.RateLimitHeaders).To(BeNil())

				updatedLimitador.Spec.RateLimitHeaders = ptr.To(limitadorv1alpha1.RateLimitHeadersType("DRAFT_VERSION_03"))
				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				updatedLimitadorDeployment := appsv1.Deployment{}
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&updatedLimitadorDeployment)).To(Succeed())
				g.Expect(updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Args).To(Equal([]string{
					"--rate-limit-headers",
					"DRAFT_VERSION_03",
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				}))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})

	Context("Reconciling command line args for telemetry", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify the limitador deployment command line args", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				updatedLimitador := limitadorv1alpha1.Limitador{}
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)).To(Succeed())

				g.Expect(updatedLimitador.Spec.Telemetry).To(BeNil())

				telemetry := limitadorv1alpha1.Telemetry("exhaustive")
				updatedLimitador.Spec.Telemetry = &telemetry
				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				updatedLimitadorDeployment := appsv1.Deployment{}
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&updatedLimitadorDeployment)).Should(Succeed())

				g.Expect(updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Args).To(
					Equal([]string{
						"--limit-name-in-labels",
						"--http-port",
						strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
						"--rls-port",
						strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
						"/home/limitador/etc/limitador-config.yaml",
						"memory",
					}))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})

	Context("Creating a new Limitador object with verbosity", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Verbosity = ptr.To(limitadorv1alpha1.VerbosityLevel(3))

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should create a new deployment with verbosity level command line arg", func(ctx SpecContext) {
			deployment := &appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"-vvv",
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				),
			)
		}, specTimeOut)
	})

	Context("Creating a new Limitador object with too high verbosity level", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		It("Should be rejected by k8s", func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Verbosity = ptr.To(limitadorv1alpha1.VerbosityLevel(6))

			Expect(k8sClient.Create(ctx, limitadorObj)).NotTo(Succeed())
		}, specTimeOut)
	})

	Context("Reconciling command line args for verbosity", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify the limitador deployment command line args", func(ctx SpecContext) {
			deployment := &appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			// verbosity level command line arg should be missing
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

			// Let's add verbosity level
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					}, &updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Verbosity = ptr.To(limitadorv1alpha1.VerbosityLevel(3))
				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newDeployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, newDeployment)).To(Succeed())

				g.Expect(newDeployment.Spec.Template.Spec.Containers[0].Args).To(Equal([]string{
					"-vvv",
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"memory",
				}))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})

	Context("Modifying limitador deployment objects", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("User tries adding side-cars to deployment CR", func(ctx SpecContext) {
			deploymentObj := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			containerObj := corev1.Container{Name: "newcontainer", Image: "someImage"}

			deploymentObj.Spec.Template.Spec.Containers = append(deploymentObj.Spec.Template.Spec.Containers, containerObj)

			Expect(k8sClient.Update(ctx, &deploymentObj)).Should(Succeed())
			updateDeploymentObj := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&updateDeploymentObj)).To(Succeed())
				g.Expect(updateDeploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})

	// This test requires actual k8s cluster
	// It's testing implementation based on CRD x-kubernetes-validations extentions
	// used to validate custom resources using Common Expression Language (CEL)
	// https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules
	Context("Disk storage does not allow multiple replicas", func() {
		It("resource is rejected", func(ctx SpecContext) {
			limitadorObj := limitadorWithInvalidDiskReplicas(testNamespace)
			err := k8sClient.Create(ctx, limitadorObj)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		}, specTimeOut)
	})

	Context("Deploying limitador object with redis storage", func() {
		var redisSecret *corev1.Secret

		BeforeEach(func(ctx SpecContext) {
			deployRedisWithContext(ctx, testNamespace)

			redisSecret = &corev1.Secret{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{Name: "redis", Namespace: testNamespace},
				StringData: map[string]string{
					"URL": fmt.Sprintf("redis://%s.%s.svc.cluster.local:6379", redisService(testNamespace).Name, testNamespace),
				},
				Type: corev1.SecretTypeOpaque,
			}

			err := k8sClient.Create(ctx, redisSecret)
			Expect(err).ToNot(HaveOccurred())

			secret := &corev1.Secret{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(redisSecret), secret)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())
		})

		It("command line is correct", func(ctx SpecContext) {
			limitadorObj := limitadorWithRedisStorage(client.ObjectKeyFromObject(redisSecret), testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			deploymentObj := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"redis",
					"$(LIMITADOR_OPERATOR_REDIS_URL)",
				),
			)
		}, specTimeOut)
	})

	Context("Deploying limitador object with redis cached storage", func() {
		var redisSecret *corev1.Secret

		BeforeEach(func(ctx SpecContext) {
			deployRedisWithContext(ctx, testNamespace)

			redisSecret = &corev1.Secret{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{Name: "redis", Namespace: testNamespace},
				StringData: map[string]string{
					"URL": fmt.Sprintf("redis://%s.%s.svc.cluster.local:6379", redisService(testNamespace).Name, testNamespace),
				},
				Type: corev1.SecretTypeOpaque,
			}

			err := k8sClient.Create(ctx, redisSecret)
			Expect(err).ToNot(HaveOccurred())

			secret := &corev1.Secret{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(redisSecret), secret)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())
		})

		It("with all defaults, the command line is correct", func(ctx SpecContext) {
			limitadorObj := limitadorWithRedisCachedStorage(client.ObjectKeyFromObject(redisSecret), testNamespace)
			limitadorObj.Spec.Storage.RedisCached.Options = nil
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			deploymentObj := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"redis_cached",
					"$(LIMITADOR_OPERATOR_REDIS_URL)",
				),
			)
		}, specTimeOut)

		It("with all the optional parameters, the command line is correct", func(ctx SpecContext) {
			limitadorObj := limitadorWithRedisCachedStorage(client.ObjectKeyFromObject(redisSecret), testNamespace)
			limitadorObj.Spec.Storage.RedisCached.Options = &limitadorv1alpha1.RedisCachedOptions{
				FlushPeriod:     ptr.To(3),
				MaxCached:       ptr.To(4),
				ResponseTimeout: ptr.To(5),
				BatchSize:       ptr.To(6),
			}

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			deploymentObj := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"redis_cached",
					"$(LIMITADOR_OPERATOR_REDIS_URL)",
					"--flush-period", "3",
					"--max-cached", "4",
					"--response-timeout", "5",
					"--batch-size", "6",
				),
			)
		}, specTimeOut)
	})

	Context("Deploying limitador object with disk storage", func() {
		It("deployment is correct", func(ctx SpecContext) {
			limitadorObj := limitadorWithDiskStorage(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			deploymentObj := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deploymentObj.Spec.Template.Spec.Volumes).To(HaveLen(2))
			Expect(deploymentObj.Spec.Template.Spec.Volumes[1]).To(
				Equal(
					corev1.Volume{
						Name: limitador.DiskVolumeName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: limitador.PVCName(limitadorObj),
								ReadOnly:  false,
							},
						},
					},
				))

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Args).To(
				HaveExactElements(
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"disk",
					limitador.DiskPath,
				),
			)
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].VolumeMounts).To(HaveLen(2))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].VolumeMounts[1]).To(
				Equal(
					corev1.VolumeMount{
						ReadOnly:  false,
						Name:      limitador.DiskVolumeName,
						MountPath: limitador.DiskPath,
					},
				),
			)
		}, specTimeOut)

		It("pvc is correct", func(ctx SpecContext) {
			limitadorObj := limitadorWithDiskStorage(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      limitador.PVCName(limitadorObj),
						Namespace: testNamespace,
					},
					pvc)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(pvc.GetOwnerReferences()).To(HaveLen(1))
		}, specTimeOut)
	})

	Context("Creating a new Limitador object with imagePullSecrets", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "regcred"}}

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should create a new deployment with imagepullsecrets", func(ctx SpecContext) {
			deployment := &appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.ImagePullSecrets).To(
				HaveExactElements(corev1.LocalObjectReference{Name: "regcred"}),
			)
		}, specTimeOut)
	})
})

func basicLimitador(ns string) *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Limitador",
			APIVersion: "limitador.kuadrant.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{GenerateName: "limitador-", Namespace: ns},
		Spec:       limitadorv1alpha1.LimitadorSpec{},
	}
}

func limitadorWithRedisStorage(redisKey client.ObjectKey, ns string) *limitadorv1alpha1.Limitador {
	l := basicLimitador(ns)
	l.Spec.Storage = &limitadorv1alpha1.Storage{
		Redis: &limitadorv1alpha1.Redis{
			ConfigSecretRef: &corev1.LocalObjectReference{
				Name: redisKey.Name,
			},
		},
	}
	return l
}

func limitadorWithRedisCachedStorage(key client.ObjectKey, ns string) *limitadorv1alpha1.Limitador {
	l := basicLimitador(ns)
	l.Spec.Storage = &limitadorv1alpha1.Storage{
		RedisCached: &limitadorv1alpha1.RedisCached{
			ConfigSecretRef: &corev1.LocalObjectReference{
				Name: key.Name,
			},
		},
	}
	return l
}

func limitadorWithDiskStorage(ns string) *limitadorv1alpha1.Limitador {
	l := basicLimitador(ns)
	l.Spec.Storage = &limitadorv1alpha1.Storage{
		Disk: &limitadorv1alpha1.DiskSpec{},
	}
	return l
}

func limitadorWithInvalidDiskReplicas(ns string) *limitadorv1alpha1.Limitador {
	l := basicLimitador(ns)
	l.Spec.Replicas = ptr.To(2)
	l.Spec.Storage = &limitadorv1alpha1.Storage{
		Disk: &limitadorv1alpha1.DiskSpec{},
	}
	return l
}

func deployRedisWithContext(ctx context.Context, ns string) {
	deployment := redisDeployment(ns)
	Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), &appsv1.Deployment{})).To(Succeed())
	}).WithContext(ctx).Should(Succeed())

	service := redisService(ns)
	Expect(k8sClient.Create(ctx, service)).Should(Succeed())
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(service), &corev1.Service{})).To(Succeed())
	}).WithContext(ctx).Should(Succeed())
}

func redisDeployment(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: ns,
			Labels:    map[string]string{"app": "redis"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "redis"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "redis"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "redis", Image: "redis"}},
				},
			},
		},
	}
}

func redisService(ns string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "redis"},
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Protocol:   corev1.ProtocolTCP,
					Port:       6379,
					TargetPort: intstr.FromInt32(6379),
				},
			},
		},
	}
}

func testLimitadorIsReady(ctx context.Context, l *limitadorv1alpha1.Limitador) func(g Gomega) {
	return func(g Gomega) {
		existing := &limitadorv1alpha1.Limitador{}
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(l), existing)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(existing.Status.Conditions, limitadorv1alpha1.StatusConditionReady)).To(BeTrue())
	}
}

func CreateNamespaceWithContext(ctx context.Context, namespace *string) {
	nsObject := &corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{GenerateName: "test-namespace-"},
	}
	Expect(k8sClient.Create(ctx, nsObject)).ToNot(HaveOccurred())

	*namespace = nsObject.Name
}

func DeleteNamespaceWithContext(ctx context.Context, namespace *string) {
	desiredTestNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: *namespace}}
	Eventually(func(g Gomega) {
		err := k8sClient.Delete(ctx, desiredTestNamespace, client.PropagationPolicy(metav1.DeletePropagationForeground))
		g.Expect(err).ToNot(BeNil())
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}).WithContext(ctx).Should(Succeed())
}
