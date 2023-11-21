package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

var (
	ExpectedDefaultImage = fmt.Sprintf("%s:%s", limitador.LimitadorRepository, "latest")
)

var _ = Describe("Limitador controller", func() {

	var testNamespace string

	BeforeEach(func() {
		CreateNamespace(&testNamespace)
	})

	AfterEach(DeleteNamespaceCallback(&testNamespace))

	Context("Creating a new basic limitador CR", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should create a Limitador service with default ports", func() {
			createdLimitadorService := v1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.ServiceName(limitadorObj),
					},
					&createdLimitadorService)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdLimitadorService.Spec.Ports).To(HaveLen(2))
			Expect(createdLimitadorService.Spec.Ports[0].Name).Should(Equal("http"))
			Expect(createdLimitadorService.Spec.Ports[0].Port).Should(Equal(limitadorv1alpha1.DefaultServiceHTTPPort))
			Expect(createdLimitadorService.Spec.Ports[1].Name).Should(Equal("grpc"))
			Expect(createdLimitadorService.Spec.Ports[1].Port).Should(Equal(limitadorv1alpha1.DefaultServiceGRPCPort))
		})

		It("Should create a new deployment with default settings", func() {
			var expectedDefaultReplicas int32 = 1
			expectedDefaultResourceRequirements := v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("250m"),
					v1.ResourceMemory: resource.MustParse("32Mi"),
				},
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500m"),
					v1.ResourceMemory: resource.MustParse("64Mi"),
				},
			}

			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&createdLimitadorDeployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

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
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Command).Should(
				// asserts no additional command line arg is added
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
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Resources).To(
				Equal(expectedDefaultResourceRequirements))
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Affinity).To(BeNil())
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
				Host: fmt.Sprintf("%s.%s.svc.cluster.local", limitador.ServiceName(limitadorObj), testNamespace),
				Ports: limitadorv1alpha1.Ports{
					GRPC: limitadorv1alpha1.DefaultServiceGRPCPort,
					HTTP: limitadorv1alpha1.DefaultServiceHTTPPort,
				},
			}))
		})

		It("Should create a ConfigMap with empty limits", func() {
			createdConfigMap := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					},
					&createdConfigMap)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(createdConfigMap.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err).ToNot(HaveOccurred())
			Expect(cmLimits).To(BeEmpty())
		})

		It("Should have not created PodDisruptionBudget", func() {
			pdb := &policyv1.PodDisruptionBudget{}
			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.PodDisruptionBudgetName(limitadorObj),
				}, pdb)
			// returns false when err is nil
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("Creating a new Limitador object with rate limit headers", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.RateLimitHeaders = &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0]

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should create a new deployment with rate limit headers command line arg", func() {
			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&createdLimitadorDeployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Command).To(
				HaveExactElements(
					"limitador-server",
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
		})
	})

	Context("Reconciling command line args for rate limit headers", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify the limitador deployment command line args", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				if err != nil {
					return false
				}

				Expect(updatedLimitador.Spec.RateLimitHeaders).To(BeNil())

				updatedLimitador.Spec.RateLimitHeaders = &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0]
				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				updatedLimitadorDeployment := appsv1.Deployment{}
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
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
						"--http-port",
						strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
						"--rls-port",
						strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
						"/home/limitador/etc/limitador-config.yaml",
						"memory",
					})
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Reconciling command line args for telemetry", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify the limitador deployment command line args", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				if err != nil {
					return false
				}

				Expect(updatedLimitador.Spec.Telemetry).To(BeNil())

				telemetry := limitadorv1alpha1.Telemetry("exhaustive")
				updatedLimitador.Spec.Telemetry = &telemetry
				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				updatedLimitadorDeployment := appsv1.Deployment{}
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&updatedLimitadorDeployment)

				if err != nil {
					return false
				}

				return reflect.DeepEqual(updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Command,
					[]string{
						"limitador-server",
						"--limit-name-in-labels",
						"--http-port",
						strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
						"--rls-port",
						strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
						"/home/limitador/etc/limitador-config.yaml",
						"memory",
					})
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Modifying limitador deployment objects", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("User tries adding side-cars to deployment CR", func() {
			deploymentObj := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			containerObj := v1.Container{Name: "newcontainer", Image: "someImage"}

			deploymentObj.Spec.Template.Spec.Containers = append(deploymentObj.Spec.Template.Spec.Containers, containerObj)

			Expect(k8sClient.Update(context.TODO(), &deploymentObj)).Should(Succeed())
			updateDeploymentObj := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
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
		It("resource is rejected", func() {
			limitadorObj := limitadorWithInvalidDiskReplicas(testNamespace)
			err := k8sClient.Create(context.TODO(), limitadorObj)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})
	})

	Context("Deploying limitador object with redis storage", func() {
		var redisSecret *v1.Secret

		BeforeEach(func() {
			deployRedis(testNamespace)

			redisSecret = &v1.Secret{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{Name: "redis", Namespace: testNamespace},
				StringData: map[string]string{
					"URL": fmt.Sprintf("redis://%s.%s.svc.cluster.local:6379", redisService(testNamespace).Name, testNamespace),
				},
				Type: v1.SecretTypeOpaque,
			}

			err := k8sClient.Create(context.Background(), redisSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				secret := &v1.Secret{}
				err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(redisSecret), secret)
				if err != nil {
					if apierrors.IsNotFound(err) {
						fmt.Fprintln(GinkgoWriter, "==== redis secret not found")
					} else {
						fmt.Fprintln(GinkgoWriter, "==== cannot read redis secret", "error", err)
					}

					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("command line is correct", func() {
			limitadorObj := limitadorWithRedisStorage(client.ObjectKeyFromObject(redisSecret), testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())

			deploymentObj := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Command).To(
				HaveExactElements(
					"limitador-server",
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"redis",
					"$(LIMITADOR_OPERATOR_REDIS_URL)",
				),
			)
		})
	})

	Context("Deploying limitador object with redis cached storage", func() {
		var redisSecret *v1.Secret

		BeforeEach(func() {
			deployRedis(testNamespace)

			redisSecret = &v1.Secret{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{Name: "redis", Namespace: testNamespace},
				StringData: map[string]string{
					"URL": fmt.Sprintf("redis://%s.%s.svc.cluster.local:6379", redisService(testNamespace).Name, testNamespace),
				},
				Type: v1.SecretTypeOpaque,
			}

			err := k8sClient.Create(context.Background(), redisSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				secret := &v1.Secret{}
				err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(redisSecret), secret)
				if err != nil {
					if apierrors.IsNotFound(err) {
						fmt.Fprintln(GinkgoWriter, "redis secret not found")
					} else {
						fmt.Fprintln(GinkgoWriter, "cannot read redis secret", "error", err)
					}

					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("command line is correct", func() {
			limitadorObj := limitadorWithRedisCachedStorage(client.ObjectKeyFromObject(redisSecret), testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())

			deploymentObj := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Command).To(
				HaveExactElements(
					"limitador-server",
					"--http-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
					"--rls-port",
					strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
					"/home/limitador/etc/limitador-config.yaml",
					"redis_cached",
					"$(LIMITADOR_OPERATOR_REDIS_URL)",
					"--ttl", "1",
					"--ratio", "2",
					"--flush-period", "3",
					"--max-cached", "4",
				),
			)
		})
	})

	Context("Deploying limitador object with disk storage", func() {
		It("deployment is correct", func() {
			limitadorObj := limitadorWithDiskStorage(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())

			deploymentObj := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deploymentObj)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deploymentObj.Spec.Template.Spec.Volumes).To(HaveLen(2))
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

			Expect(deploymentObj.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploymentObj.Spec.Template.Spec.Containers[0].Command).To(
				HaveExactElements(
					"limitador-server",
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
					v1.VolumeMount{
						ReadOnly:  false,
						Name:      limitador.DiskVolumeName,
						MountPath: limitador.DiskPath,
					},
				),
			)
		})

		It("pvc is correct", func() {
			limitadorObj := limitadorWithDiskStorage(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())

			pvc := &v1.PersistentVolumeClaim{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      limitador.PVCName(limitadorObj),
						Namespace: testNamespace,
					},
					pvc)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(pvc.GetOwnerReferences()).To(HaveLen(1))
		})
	})
})

func basicLimitador(ns string) *limitadorv1alpha1.Limitador {
	// The name can't start with a number.
	name := "a" + string(uuid.NewUUID())

	return &limitadorv1alpha1.Limitador{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Limitador",
			APIVersion: "limitador.kuadrant.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       limitadorv1alpha1.LimitadorSpec{},
	}
}

func limitadorWithRedisStorage(redisKey client.ObjectKey, ns string) *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-redis-storage", Namespace: ns},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Storage: &limitadorv1alpha1.Storage{
				Redis: &limitadorv1alpha1.Redis{
					ConfigSecretRef: &v1.LocalObjectReference{
						Name: redisKey.Name,
					},
				},
			},
		},
	}
}

func limitadorWithRedisCachedStorage(key client.ObjectKey, ns string) *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-redis-cached-storage", Namespace: ns},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Storage: &limitadorv1alpha1.Storage{
				RedisCached: &limitadorv1alpha1.RedisCached{
					ConfigSecretRef: &v1.LocalObjectReference{
						Name: key.Name,
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

func limitadorWithDiskStorage(ns string) *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-disk-storage", Namespace: ns},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Storage: &limitadorv1alpha1.Storage{
				Disk: &limitadorv1alpha1.DiskSpec{},
			},
		},
	}
}

func limitadorWithInvalidDiskReplicas(ns string) *limitadorv1alpha1.Limitador {
	return &limitadorv1alpha1.Limitador{
		TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "limitador-with-invalid-disk-replicas", Namespace: ns},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Replicas: &[]int{2}[0],
			Storage: &limitadorv1alpha1.Storage{
				Disk: &limitadorv1alpha1.DiskSpec{},
			},
		},
	}
}

func deployRedis(ns string) {
	deployment := redisDeployment(ns)
	Expect(k8sClient.Create(context.TODO(), deployment)).Should(Succeed())
	Eventually(func() bool {
		d := &appsv1.Deployment{}
		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(deployment), d)
		return err == nil
	}, timeout, interval).Should(BeTrue())

	service := redisService(ns)
	Expect(k8sClient.Create(context.TODO(), service)).Should(Succeed())
	Eventually(func() bool {
		s := &v1.Service{}
		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(service), s)
		return err == nil
	}, timeout, interval).Should(BeTrue())
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

func redisService(ns string) *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: ns,
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

func testLimitadorIsReady(l *limitadorv1alpha1.Limitador) func() bool {
	return func() bool {
		existing := &limitadorv1alpha1.Limitador{}
		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(l), existing)
		return err == nil && meta.IsStatusConditionTrue(existing.Status.Conditions, "Ready")
	}
}

func CreateNamespace(namespace *string) {
	var generatedTestNamespace = "test-namespace-" + string(uuid.NewUUID())

	nsObject := &v1.Namespace{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{Name: generatedTestNamespace},
	}

	err := k8sClient.Create(context.Background(), nsObject)
	Expect(err).ToNot(HaveOccurred())

	existingNamespace := &v1.Namespace{}
	Eventually(func() bool {
		err := k8sClient.Get(context.Background(), types.NamespacedName{Name: generatedTestNamespace}, existingNamespace)
		return err == nil
	}, time.Minute, 5*time.Second).Should(BeTrue())

	*namespace = existingNamespace.Name
}

func DeleteNamespaceCallback(namespace *string) func() {
	return func() {
		desiredTestNamespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: *namespace}}
		err := k8sClient.Delete(context.Background(), desiredTestNamespace, client.PropagationPolicy(metav1.DeletePropagationForeground))

		Expect(err).ToNot(HaveOccurred())

		existingNamespace := &v1.Namespace{}
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: *namespace}, existingNamespace)
			if err != nil && apierrors.IsNotFound(err) {
				return true
			}
			return false
		}, 3*time.Minute, 2*time.Second).Should(BeTrue())
	}
}
