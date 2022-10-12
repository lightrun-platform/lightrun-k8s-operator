package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	agentv1beta "github.com/lightrun-platform/lightrun-k8s-operator/api/v1beta"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1ac "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	cmNamePrefix      = "lightrunagent-cm-"
	cmVolumeName      = "lightrunagent-config"
	initContainerName = "lightrun-installer"
)

func (r *LightrunJavaAgentReconciler) createAgentConfig(lightrunJavaAgent *agentv1beta.LightrunJavaAgent) (corev1.ConfigMap, error) {
	populateTags(lightrunJavaAgent.Spec.AgentTags, lightrunJavaAgent.Spec.AgentName, &metadata)
	jsonString, err := json.Marshal(metadata)
	if err != nil {
		return corev1.ConfigMap{}, err
	}
	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      (cmNamePrefix + lightrunJavaAgent.Name),
			Namespace: lightrunJavaAgent.Namespace,
		},
		Data: map[string]string{
			"config":   parseAgentConfig(lightrunJavaAgent.Spec.AgentConfig),
			"metadata": string(jsonString),
		},
	}

	if err := ctrl.SetControllerReference(lightrunJavaAgent, &configMap, r.Scheme); err != nil {
		return configMap, err
	}
	return configMap, nil
}

func (r *LightrunJavaAgentReconciler) patchDeployment(ctx context.Context, lightrunJavaAgent *agentv1beta.LightrunJavaAgent, secret *corev1.Secret, origDeployment *appsv1.Deployment, deploymentApplyConfig *appsv1ac.DeploymentApplyConfiguration, cmDataHash uint64) error {

	// init spec.template.spec
	deploymentApplyConfig.WithSpec(
		appsv1ac.DeploymentSpec().WithTemplate(
			corev1ac.PodTemplateSpec().WithSpec(
				corev1ac.PodSpec(),
			).WithAnnotations(map[string]string{
				"lightrun.com/configmap-hash": fmt.Sprint(cmDataHash),
			},
			),
		),
	).WithAnnotations(map[string]string{
		"lightrun.com/lightrunjavaagent": lightrunJavaAgent.Name,
	})
	r.addVolume(deploymentApplyConfig, lightrunJavaAgent)
	r.addInitContainer(deploymentApplyConfig, lightrunJavaAgent, secret)
	err = r.patchAppContainers(lightrunJavaAgent, origDeployment, deploymentApplyConfig)
	if err != nil {
		return err
	}
	return nil
}

func (r *LightrunJavaAgentReconciler) addVolume(deploymentApplyConfig *appsv1ac.DeploymentApplyConfiguration, lightrunJavaAgent *agentv1beta.LightrunJavaAgent) {

	deploymentApplyConfig.Spec.Template.Spec.
		WithVolumes(
			corev1ac.Volume().
				WithName(lightrunJavaAgent.Spec.InitContainer.SharedVolumeName).
				WithEmptyDir(
					corev1ac.EmptyDirVolumeSource(),
				),
		).WithVolumes(
		corev1ac.Volume().
			WithName(cmVolumeName).
			WithConfigMap(
				corev1ac.ConfigMapVolumeSource().
					WithName(cmNamePrefix+lightrunJavaAgent.Name).
					WithItems(
						corev1ac.KeyToPath().WithKey("config").WithPath("agent.config"),
						corev1ac.KeyToPath().WithKey("metadata").WithPath("agent.metadata.json"),
					),
			),
	)
}

func (r *LightrunJavaAgentReconciler) addInitContainer(deploymentApplyConfig *appsv1ac.DeploymentApplyConfiguration, lightrunJavaAgent *agentv1beta.LightrunJavaAgent, secret *corev1.Secret) {

	deploymentApplyConfig.Spec.Template.Spec.WithInitContainers(
		corev1ac.Container().
			WithName(initContainerName).
			WithImage(lightrunJavaAgent.Spec.InitContainer.Image).
			WithVolumeMounts(
				corev1ac.VolumeMount().WithName(lightrunJavaAgent.Spec.InitContainer.SharedVolumeName).WithMountPath("/tmp/"),
				corev1ac.VolumeMount().WithName(cmVolumeName).WithMountPath("/tmp/cm/"),
			).WithEnv(
			corev1ac.EnvVar().WithName("LIGHTRUN_KEY").WithValueFrom(
				corev1ac.EnvVarSource().WithSecretKeyRef(
					corev1ac.SecretKeySelector().WithName(secret.Name).WithKey("lightrun_key"),
				),
			),
			corev1ac.EnvVar().WithName("PINNED_CERT").WithValueFrom(
				corev1ac.EnvVarSource().WithSecretKeyRef(
					corev1ac.SecretKeySelector().WithName(secret.Name).WithKey("pinned_cert_hash"),
				),
			),
			corev1ac.EnvVar().WithName("LIGHTRUN_SERVER").WithValue(lightrunJavaAgent.Spec.ServerHostname),
		).
			WithResources(
				corev1ac.ResourceRequirements().
					WithLimits(
						corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(50), resource.BinarySI),
							corev1.ResourceMemory: *resource.NewScaledQuantity(int64(64), resource.Scale(6)), // 500 * 10^6 = 500M
						},
					).WithRequests(
					corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(50), resource.BinarySI),
						corev1.ResourceMemory: *resource.NewScaledQuantity(int64(64), resource.Scale(6)),
					},
				),
			),
	)
}

func (r *LightrunJavaAgentReconciler) patchAppContainers(lightrunJavaAgent *agentv1beta.LightrunJavaAgent, origDeployment *appsv1.Deployment, deploymentApplyConfig *appsv1ac.DeploymentApplyConfiguration) error {
	var found bool = false
	for _, container := range origDeployment.Spec.Template.Spec.Containers {
		for _, targetContainer := range lightrunJavaAgent.Spec.ContainerSelector {
			if targetContainer == container.Name {
				found = true
				deploymentApplyConfig.Spec.Template.Spec.WithContainers(
					corev1ac.Container().
						WithName(container.Name).
						WithImage(container.Image).
						WithVolumeMounts(
							corev1ac.VolumeMount().WithMountPath(lightrunJavaAgent.Spec.InitContainer.SharedVolumeMountPath).WithName(lightrunJavaAgent.Spec.InitContainer.SharedVolumeName),
						),
				)
			}
		}
	}
	if !found {
		err = errors.New("unable to find matching container to patch")
		return err
	}
	return nil
}

// Client side patch, as we can't update value from 2 sources
func (r *LightrunJavaAgentReconciler) patchJavaToolEnv(container *corev1.Container, targetEnvVar string, mountPath string) error {
	agentArg := "-agentpath:" + mountPath + "/agent/lightrun_agent.so"

	javaToolOptionsIndex := -1

	for index, envVar := range container.Env {
		if envVar.Name == targetEnvVar {
			javaToolOptionsIndex = index
			break
		}
	}

	if javaToolOptionsIndex == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  targetEnvVar,
			Value: agentArg,
		})
	} else {
		if !strings.Contains(container.Env[javaToolOptionsIndex].Value, agentArg) {
			if len(container.Env[javaToolOptionsIndex].Value+" "+agentArg) > 1024 {
				return errors.New(targetEnvVar + " has more that 1024 chars")
			}
			container.Env[javaToolOptionsIndex].Value = container.Env[javaToolOptionsIndex].Value + " " + agentArg
		}
	}
	return nil
}

func (r *LightrunJavaAgentReconciler) unpatchJavaToolEnv(container *corev1.Container, targetEnvVar string, mountPath string) *corev1.Container {
	agentArg := "-agentpath:" + mountPath + "/agent/lightrun_agent.so"
	var updatedSlice []corev1.EnvVar
	for _, envVar := range container.Env {
		if envVar.Name == targetEnvVar {
			var updatedEnv []string
			optArray := strings.Split(envVar.Value, " ")
			for _, opt := range optArray {
				if opt != agentArg {
					updatedEnv = append(updatedEnv, opt)
				}
			}
			if len(updatedEnv) > 0 {
				envVar.Value = strings.Join(updatedEnv, " ")
				updatedSlice = append(updatedSlice, envVar)
			}
		} else {
			updatedSlice = append(updatedSlice, envVar)
		}
	}
	container.Env = updatedSlice
	return container
}
