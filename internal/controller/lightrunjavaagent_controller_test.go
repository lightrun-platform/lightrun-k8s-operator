package controller

import (
	"context"
	"time"

	agentsv1beta "github.com/lightrun-platform/lightrun-k8s-operator/api/v1beta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("LightrunJavaAgent controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		lragent1Name       = "lragent"
		deployment         = "app-deployment"
		secret             = "agent-secret"
		server             = "example.lightrun.com"
		agentName          = "coolio-agent"
		timeout            = time.Second * 10
		duration           = time.Second * 10
		interval           = time.Millisecond * 250
		wrongNamespace     = "wrong-namespace"
		initContainerImage = "lightruncom/lightrun-init-agent:latest"
		agentPlatform      = "linux"
		initVolumeName     = "lightrun-agent-init"
		javaEnv            = "JAVA_TOOL_OPTIONS"
		defaultAgentPath   = "-agentpath:/lightrun/agent/lightrun_agent.so"
		agentCliFlags      = "--lightrun_extra_class_path=<PATH_TO_JAR>"
	)
	var containerSelector = []string{"app", "app2"}
	var agentConfig map[string]string = map[string]string{"max_log_cpu_cost": "2"}
	var agentTags []string = []string{"new_tag", "prod"}
	var secretData map[string]string = map[string]string{
		"LIGHTRUN_KEY":     "some_key",
		"LIGHTRUN_COMPANY": "some_company",
	}

	var patchedDepl appsv1.Deployment
	deplRequest := types.NamespacedName{
		Name:      deployment,
		Namespace: testNamespace,
	}

	var patchedDepl2 appsv1.Deployment
	deplRequest2 := types.NamespacedName{
		Name:      deployment + "-2",
		Namespace: testNamespace,
	}

	var patchedDepl3 appsv1.Deployment
	deplRequest3 := types.NamespacedName{
		Name:      deployment + "-3",
		Namespace: wrongNamespace,
	}

	var cm corev1.ConfigMap
	cmRequest := types.NamespacedName{
		Name:      cmNamePrefix + lragent1Name,
		Namespace: testNamespace,
	}

	var lrAgent agentsv1beta.LightrunJavaAgent
	lrAgentRequest := types.NamespacedName{
		Name:      lragent1Name,
		Namespace: testNamespace,
	}

	var lrAgent2 agentsv1beta.LightrunJavaAgent
	lrAgentRequest2 := types.NamespacedName{
		Name:      "lragent2",
		Namespace: testNamespace,
	}

	var lrAgent3 agentsv1beta.LightrunJavaAgent
	lrAgentRequest3 := types.NamespacedName{
		Name:      "duplicate",
		Namespace: testNamespace,
	}

	var lrAgent4 agentsv1beta.LightrunJavaAgent
	lrAgentRequest4 := types.NamespacedName{
		Name:      "wrong-namespace",
		Namespace: wrongNamespace,
	}

	ctx := context.Background()
	Context("When setting up the test environment", func() {
		It("Should create a test Namespace", func() {
			By("Creating a Namespace")
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, &ns)).Should(Succeed())
		})
		It("Should create a wrong Namespace", func() {
			By("Creating a Namespace")
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: wrongNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, &ns)).Should(Succeed())
		})
		It("Should create LightrunJavaAgent custom resource", func() {
			By("Creating a first LightrunJavaAgent resource")
			lrAgent := agentsv1beta.LightrunJavaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      lragent1Name,
					Namespace: testNamespace,
				},
				Spec: agentsv1beta.LightrunJavaAgentSpec{
					DeploymentName:    deployment,
					SecretName:        secret,
					ServerHostname:    server,
					AgentName:         agentName,
					AgentTags:         agentTags,
					AgentConfig:       agentConfig,
					AgentCliFlags:     agentCliFlags,
					AgentEnvVarName:   javaEnv,
					ContainerSelector: containerSelector,
					InitContainer: agentsv1beta.InitContainer{
						Image:                 initContainerImage,
						SharedVolumeName:      initVolumeName,
						SharedVolumeMountPath: "/lightrun",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &lrAgent)).Should(Succeed())

			By("Creating a first LightrunJavaAgent resource")
			lrAgent2 := agentsv1beta.LightrunJavaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lragent2",
					Namespace: testNamespace,
				},
				Spec: agentsv1beta.LightrunJavaAgentSpec{
					DeploymentName:    deployment + "-2",
					SecretName:        secret,
					ServerHostname:    server,
					AgentName:         agentName,
					AgentTags:         agentTags,
					AgentConfig:       agentConfig,
					AgentEnvVarName:   javaEnv,
					ContainerSelector: containerSelector,
					InitContainer: agentsv1beta.InitContainer{
						Image:                 initContainerImage,
						SharedVolumeName:      initVolumeName,
						SharedVolumeMountPath: "/lightrun",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &lrAgent2)).Should(Succeed())

			By("Creating a secret")
			secret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret,
					Namespace: testNamespace,
				},
				StringData: secretData,
			}
			Expect(k8sClient.Create(ctx, &secret)).Should(Succeed())

		})
	})

	It("Should create Deployment", func() {
		By("Creating deployment")
		ctx := context.Background()

		depl := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment,
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "app"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "app"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "busybox",
							},
							{
								Name:  "app2",
								Image: "busybox",
								Env: []corev1.EnvVar{
									{
										Name:  javaEnv,
										Value: "-Djava.net.preferIPv4Stack=true",
									},
								},
							},
							{
								Name:  "no-patch",
								Image: "busybox",
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, &depl)).Should(Succeed())
	})

	Context("When patching deployment matched by CRD", func() {
		It("Should add init Container", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				if len(patchedDepl.Spec.Template.Spec.InitContainers) != 0 {
					return true
				}
				return false
			}).Should(BeTrue())
		})

		It("Should patch  Env Vars of containers with agentCliFlags value", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				for _, container := range patchedDepl.Spec.Template.Spec.Containers {
					for _, envVar := range container.Env {
						if envVar.Name == javaEnv {
							if container.Name == "app" {
								if envVar.Value != defaultAgentPath+"="+agentCliFlags {
									return false
								}
							} else if container.Name == "app2" {
								if envVar.Value != "-Djava.net.preferIPv4Stack=true "+defaultAgentPath+"="+agentCliFlags {
									return false
								}
							}
						}
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("Should add VolumeMount to Containers", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				var flag int
				for _, container := range patchedDepl.Spec.Template.Spec.Containers {
					flag = -1
					if container.Name != "no-patch" {
						for _, volume := range container.VolumeMounts {
							if volume.Name == initVolumeName {
								flag = 1
							}
						}
						if flag == -1 {
							return false
						}
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("Should not patch 3rd container that not mentioned in CRD", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				for _, container := range patchedDepl.Spec.Template.Spec.Containers {
					if container.Name == "no-patch" {
						for _, envVar := range container.Env {
							if envVar.Name == javaEnv {
								return false
							}
						}
						for _, volume := range container.VolumeMounts {
							if volume.Name == initVolumeName {
								return false
							}
						}
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("Should add volumes to the deployment", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				var desiredVolumes int = 0
				for _, volume := range patchedDepl.Spec.Template.Spec.Volumes {
					if volume.Name == initVolumeName || volume.Name == cmVolumeName {
						desiredVolumes += 1
					}
				}
				return desiredVolumes == 2
			}).Should(BeTrue())
		})

		It("Should create config map", func() {
			Expect(k8sClient.Get(ctx, cmRequest, &cm)).Should(Succeed())
		})

		It("Should add annotations to deployment", func() {
			Eventually(func() bool {
				flag := 0
				for k, v := range patchedDepl.ObjectMeta.Annotations {
					if k == "lightrun.com/lightrunjavaagent" && v == lragent1Name {
						flag += 1
					}
				}
				for k := range patchedDepl.Spec.Template.Annotations {
					if k == "lightrun.com/configmap-hash" {
						flag += 1
					}
				}
				return flag == 2
			}).Should(BeTrue())
		})

		It("Should add finalizer to first CRD", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest, &lrAgent); err != nil {
					return false
				}
				return len(lrAgent.ObjectMeta.Finalizers) != 0
			}).Should(BeTrue())
		})

		It("Should not add finalizer to second CRD", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest2, &lrAgent2); err != nil {
					return false
				}
				return len(lrAgent2.ObjectMeta.Finalizers) == 0
			}).Should(BeTrue())
		})
	})

	Context("When deleting first CRD", func() {
		It("Should delete CRD", func() {
			lrAgent := agentsv1beta.LightrunJavaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lragent",
					Namespace: testNamespace,
				},
			}
			Expect(k8sClient.Delete(ctx, &lrAgent)).Should(Succeed())
		})

		It("Should remove volumes from the deployment", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				for _, volume := range patchedDepl.Spec.Template.Spec.Volumes {
					if volume.Name == initVolumeName || volume.Name == cmVolumeName {
						return false
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("Should remove init container from the deployment", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				return len(patchedDepl.Spec.Template.Spec.InitContainers) == 0
			}).Should(BeTrue())
		})

		It("Should rollback "+javaEnv+" env var", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				for _, container := range patchedDepl.Spec.Template.Spec.Containers {
					if container.Name == "no-patch" {
						continue
					}
					for _, envVar := range container.Env {
						if container.Name == "app" {
							if envVar.Name == javaEnv {
								return false
							}
						} else if container.Name == "app2" {
							if envVar.Name == javaEnv {
								if envVar.Value != "-Djava.net.preferIPv4Stack=true" {
									//logger.Info("second container", envVar.Name, envVar.Value)
									return false
								}
							}
						}
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("Should delete config map", func() {
			Expect(k8sClient.Get(ctx, cmRequest, &cm)).Error()
		})

		It("Should delete annotations from deployment", func() {
			Eventually(func() bool {
				for k := range patchedDepl.ObjectMeta.Annotations {
					if k == "lightrun.com/lightrunjavaagent" {
						return false
					}
				}
				for k := range patchedDepl.Spec.Template.Annotations {
					if k == "lightrun.com/configmap-hash" {
						return false
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("Should remove Volume mounts from containers in the deployment", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest, &patchedDepl); err != nil {
					return false
				}
				for _, container := range patchedDepl.Spec.Template.Spec.Containers {
					if container.Name != "no-patch" {
						if len(container.VolumeMounts) != 0 {
							return false
						}
					}
				}
				return true
			}).Should(BeTrue())
		})

		//TODO: status update
	})

	// Create and delete deployment matching  2nd CRD to check that finalizer was removed
	Context("When deleting deployment before removing CRD", func() {
		It("prepare deployment for 2nd CRD", func() {
			By("Creating deployment")
			depl := appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployment + "-2",
					Namespace: testNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "app"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "app"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "app",
									Image: "busybox",
								},
								{
									Name:  "app2",
									Image: "busybox",
									Env: []corev1.EnvVar{
										{
											Name:  javaEnv,
											Value: "-Djava.net.preferIPv4Stack=true",
										},
									},
								},
								{
									Name:  "no-patch",
									Image: "busybox",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &depl)).Should(Succeed())
		})

		It("Should add finalizer to second CRD", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest2, &lrAgent2); err != nil {
					return false
				}
				return len(lrAgent2.ObjectMeta.Finalizers) != 0
			}).Should(BeTrue())
		})

		It("Should patch  Env Vars of containers with default agent path", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest2, &patchedDepl2); err != nil {
					return false
				}
				for _, container := range patchedDepl2.Spec.Template.Spec.Containers {
					for _, envVar := range container.Env {
						if envVar.Name == javaEnv {
							if container.Name == "app" {
								if envVar.Value != defaultAgentPath {
									return false
								}
							} else if container.Name == "app2" {
								if envVar.Value != "-Djava.net.preferIPv4Stack=true "+defaultAgentPath {
									return false
								}
							}
						}
					}
				}
				return true
			}).Should(BeTrue())
		})

		It("Should delete deployment", func() {
			depl := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployment + "-2",
					Namespace: testNamespace,
				},
			}
			Expect(k8sClient.Delete(ctx, &depl)).Should(Succeed())
		})

		It("Should remove finalizer from the second CRD", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest2, &lrAgent2); err != nil {
					return false
				}
				return len(lrAgent2.ObjectMeta.Finalizers) == 0
			}).Should(BeTrue())
		})

	})

	Context("When creating CR with deployment already patched by another CR", func() {
		It("Should create Deployment", func() {
			By("Creating deployment")
			depl := appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployment + "-2",
					Namespace: testNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "app"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "app"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "app",
									Image: "busybox",
								},
								{
									Name:  "app2",
									Image: "busybox",
									Env: []corev1.EnvVar{
										{
											Name:  javaEnv,
											Value: "-Djava.net.preferIPv4Stack=true",
										},
									},
								},
								{
									Name:  "no-patch",
									Image: "busybox",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &depl)).Should(Succeed())
		})

		It("Should have successful status of existing CR", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest2, &lrAgent2); err != nil {
					return false
				}
				return lrAgent2.Status.DeploymentStatus == "Ready"
			}).Should(BeTrue())
		})

		It("prepare new CR with patched deployment", func() {
			By("Creating new CR")
			lrAgent3 := agentsv1beta.LightrunJavaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate",
					Namespace: testNamespace,
				},
				Spec: agentsv1beta.LightrunJavaAgentSpec{
					DeploymentName:    deployment + "-2",
					SecretName:        secret,
					ServerHostname:    server,
					AgentName:         agentName,
					AgentTags:         agentTags,
					AgentConfig:       agentConfig,
					AgentEnvVarName:   javaEnv,
					ContainerSelector: containerSelector,
					InitContainer: agentsv1beta.InitContainer{
						Image:                 initContainerImage,
						SharedVolumeName:      initVolumeName,
						SharedVolumeMountPath: "/lightrun",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &lrAgent3)).Should(Succeed())
		})

		It("Should have failed status of CR", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest3, &lrAgent3); err != nil {
					return false
				}
				return lrAgent3.Status.DeploymentStatus == "ReconcileFailed"
			}).Should(BeTrue())
		})
		It("Should not add finalizer to the duplicate CR", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest3, &lrAgent3); err != nil {
					return false
				}
				return len(lrAgent3.ObjectMeta.Finalizers) == 0
			}).Should(BeTrue())
		})

		It("Should keep deployment annotation of the original CR", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, deplRequest2, &patchedDepl2); err != nil {
					return false
				}
				return patchedDepl2.Annotations["lightrun.com/lightrunjavaagent"] == lrAgent2.Name
			}).Should(BeTrue())
		})

	})
	Context("When trying to patch deployment in the wrong namespace ", func() {
		It("Should create Deployment", func() {
			By("Creating deployment")
			depl := appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployment + "-3",
					Namespace: wrongNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "app"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "app"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "app",
									Image: "busybox",
								},
								{
									Name:  "app2",
									Image: "busybox",
									Env: []corev1.EnvVar{
										{
											Name:  javaEnv,
											Value: "-Djava.net.preferIPv4Stack=true",
										},
									},
								},
								{
									Name:  "no-patch",
									Image: "busybox",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &depl)).Should(Succeed())
		})
		It("Should create CR in the wrong namespace", func() {
			By("Creating new CR")
			lrAgent4 := agentsv1beta.LightrunJavaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "wrong-namespace",
					Namespace: wrongNamespace,
				},
				Spec: agentsv1beta.LightrunJavaAgentSpec{
					DeploymentName:    deployment + "-3",
					SecretName:        secret,
					ServerHostname:    server,
					AgentName:         agentName,
					AgentTags:         agentTags,
					AgentConfig:       agentConfig,
					AgentEnvVarName:   javaEnv,
					ContainerSelector: containerSelector,
					InitContainer: agentsv1beta.InitContainer{
						Image:                 initContainerImage,
						SharedVolumeName:      initVolumeName,
						SharedVolumeMountPath: "/lightrun",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &lrAgent4)).Should(Succeed())
		})
		It("Should not change the CR status", func() {
			Consistently(func() bool {
				if err := k8sClient.Get(ctx, lrAgentRequest4, &lrAgent4); err != nil {
					return false
				}
				return lrAgent4.Status.DeploymentStatus == "" && lrAgent4.Status.Conditions == nil
			}).Should(BeTrue())
		})
		It("Should not patch the deployment", func() {
			Consistently(func() bool {
				if err := k8sClient.Get(ctx, deplRequest3, &patchedDepl3); err != nil {
					return false
				}
				if _, ok := patchedDepl3.Annotations["lightrun.com/lightrunjavaagent"]; !ok && len(patchedDepl3.Finalizers) == 0 {
					return true
				}
				return false
			}).Should(BeTrue())
		})
	})
})
