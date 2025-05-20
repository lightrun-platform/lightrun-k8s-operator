/*
Copyright 2022 Lightrun

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1ac "k8s.io/client-go/applyconfigurations/apps/v1"

	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/go-logr/logr"
	agentv1beta "github.com/lightrun-platform/lightrun-k8s-operator/api/v1beta"
)

const (
	deploymentNameIndexField = "spec.deployment"
	workloadNameIndexField   = "spec.workloadName"
	secretNameIndexField     = "spec.secret"
	finalizerName            = "agent.finalizers.lightrun.com"
)

var err error
var secret *corev1.Secret

// LightrunJavaAgentReconciler reconciles a LightrunJavaAgent object
type LightrunJavaAgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=agents.lightrun.com,resources=lightrunjavaagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agents.lightrun.com,resources=lightrunjavaagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agents.lightrun.com,resources=lightrunjavaagents/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;watch;list;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;watch;list;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;watch;list

func (r *LightrunJavaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("lightrunJavaAgent", req.NamespacedName)
	lightrunJavaAgent := &agentv1beta.LightrunJavaAgent{}
	if err = r.Get(ctx, req.NamespacedName, lightrunJavaAgent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Determine which workload type to reconcile
	workloadType, err := r.determineWorkloadType(lightrunJavaAgent)
	if err != nil {
		log.Error(err, "failed to determine workload type")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}
	switch workloadType {
	case agentv1beta.WorkloadTypeDeployment:
		return r.reconcileDeployment(ctx, lightrunJavaAgent, req.Namespace)
	case agentv1beta.WorkloadTypeStatefulSet:
		return r.reconcileStatefulSet(ctx, lightrunJavaAgent, req.Namespace)
	default:
		return r.errorStatus(ctx, lightrunJavaAgent, fmt.Errorf("unsupported workload type: %s", workloadType))
	}
}

func (r *LightrunJavaAgentReconciler) determineWorkloadType(lightrunJavaAgent *agentv1beta.LightrunJavaAgent) (agentv1beta.WorkloadType, error) {
	spec := lightrunJavaAgent.Spec
	// Check which configuration approach is being used
	var isDeploymentConfigured bool = spec.DeploymentName != ""
	var isWorkloadConfigured bool = spec.WorkloadName != "" || spec.WorkloadType != ""

	// === Case 1: Legacy only — DeploymentName only ===
	if isDeploymentConfigured && !isWorkloadConfigured {
		r.Log.Info("Using deprecated field deploymentName, consider migrating to workloadName and workloadType")
		return agentv1beta.WorkloadTypeDeployment, nil
	}

	// === Case 2: New fields — WorkloadName + WorkloadType ===
	if !isDeploymentConfigured && isWorkloadConfigured {
		if spec.WorkloadType == "" {
			return "", errors.New("WorkloadType must be set when using WorkloadName")
		}
		if spec.WorkloadName == "" {
			return "", errors.New("WorkloadName must be set when using WorkloadType")
		}
		return spec.WorkloadType, nil
	}

	// === Case 3: Misconfigured — Both fields exists or both are empty ===
	if isDeploymentConfigured && isWorkloadConfigured {
		return "", errors.New("invalid configuration: use either deploymentName (legacy) OR workloadName with workloadType, not both")
	}

	// === Case 4: Fully empty or malformed ===
	return "", errors.New("invalid configuration: must set either DeploymentName (legacy) or WorkloadName with WorkloadType")
}

// reconcileDeployment handles the reconciliation logic for Deployment workloads
func (r *LightrunJavaAgentReconciler) reconcileDeployment(ctx context.Context, lightrunJavaAgent *agentv1beta.LightrunJavaAgent, namespace string) (ctrl.Result, error) {
	// Get the workload name - use DeploymentName for backward compatibility
	// or WorkloadName for newer CR versions
	deploymentName := lightrunJavaAgent.Spec.WorkloadName
	if deploymentName == "" && lightrunJavaAgent.Spec.DeploymentName != "" {
		// Fall back to legacy field if WorkloadName isn't set
		deploymentName = lightrunJavaAgent.Spec.DeploymentName
	}

	log := r.Log.WithValues("lightrunJavaAgent", lightrunJavaAgent.Name, "deployment", deploymentName)
	fieldManager := "lightrun-conrtoller"

	deplNamespacedObj := client.ObjectKey{
		Name:      deploymentName,
		Namespace: namespace,
	}
	originalDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, deplNamespacedObj, originalDeployment)
	if err != nil {
		// Deployment not found
		if client.IgnoreNotFound(err) == nil {
			log.Info("Deployment not found. Verify name/namespace", "Deployment", deploymentName)
			// remove our finalizer from the list and update it.
			err = r.removeFinalizer(ctx, lightrunJavaAgent, finalizerName)
			if err != nil {
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}
			return r.errorStatus(ctx, lightrunJavaAgent, errors.New("deployment not found"))
		} else {
			log.Error(err, "unable to fetch deployment")
			return r.errorStatus(ctx, lightrunJavaAgent, err)
		}
	}

	if oldLrjaName, ok := originalDeployment.Annotations[annotationAgentName]; ok && oldLrjaName != lightrunJavaAgent.Name {
		log.Error(err, "Deployment already patched by LightrunJavaAgent", "Existing LightrunJavaAgent", oldLrjaName)
		return r.errorStatus(ctx, lightrunJavaAgent, errors.New("deployment already patched"))
	}

	deploymentApplyConfig, err := appsv1ac.ExtractDeployment(originalDeployment, fieldManager)
	if err != nil {
		log.Error(err, "failed to extract Deployment")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Finalization
	if lightrunJavaAgent.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted

		log.V(2).Info("Searching for secret", "Name", lightrunJavaAgent.Spec.SecretName)
		secretNamespacedObj := client.ObjectKey{
			Name:      lightrunJavaAgent.Spec.SecretName,
			Namespace: namespace,
		}
		secret = &corev1.Secret{}
		err = r.Get(ctx, secretNamespacedObj, secret)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				log.Error(err, "Secret not found", "Secret", lightrunJavaAgent.Spec.SecretName)
			}
			return r.errorStatus(ctx, lightrunJavaAgent, err)
		}

		// Ensure that finalizer is in place
		if !containsString(lightrunJavaAgent.ObjectMeta.Finalizers, finalizerName) {
			log.Info("Start working on deployment", "Deployment", deploymentName)
			log.Info("Adding finalizer")
			err = r.addFinalizer(ctx, lightrunJavaAgent, finalizerName)
			if err != nil {
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}
		}
	} else {
		// The object is being deleted
		log.Info("LightrunJavaAgent is being deleted", "lightrunJavaAgent", lightrunJavaAgent.Name)
		// Unpatch deployment
		if containsString(lightrunJavaAgent.ObjectMeta.Finalizers, finalizerName) {
			log.Info("Unpatching deployment", "Deployment", originalDeployment.Name)

			// Remove agent from JAVA_TOOL_OPTIONS. Client side patch
			clientSidePatch := client.MergeFrom(originalDeployment.DeepCopy())
			for i, container := range originalDeployment.Spec.Template.Spec.Containers {
				for _, targetContainer := range lightrunJavaAgent.Spec.ContainerSelector {
					if targetContainer == container.Name {
						r.unpatchJavaToolEnv(originalDeployment.Annotations, &originalDeployment.Spec.Template.Spec.Containers[i])
					}
				}

			}
			delete(originalDeployment.Annotations, annotationPatchedEnvName)
			delete(originalDeployment.Annotations, annotationPatchedEnvValue)
			err = r.Patch(ctx, originalDeployment, clientSidePatch)
			if err != nil {
				log.Error(err, "unable to unpatch "+lightrunJavaAgent.Spec.AgentEnvVarName)
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}

			// Remove Volumes and init container
			emptyApplyConfig := appsv1ac.Deployment(deplNamespacedObj.Name, deplNamespacedObj.Namespace)
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(emptyApplyConfig)
			if err != nil {
				log.Error(err, "failed to convert Deployment to unstructured")
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}
			patch := &unstructured.Unstructured{
				Object: obj,
			}
			err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
				FieldManager: fieldManager,
				Force:        pointer.Bool(true),
			})
			if err != nil {
				log.Error(err, "failed to unpatch deployment")
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}

			// remove our finalizer from the list and update it.
			log.Info("Removing finalizer")
			err = r.removeFinalizer(ctx, lightrunJavaAgent, finalizerName)
			if err != nil {
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}

			log.Info("Deployment returned to original state", "Deployment", deploymentName)
			return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeProgressing)
		} else {
			// Nothing to do here
			return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeProgressing)
		}
	}

	// Verify that env var won't exceed 1024 chars
	agentArg, err := agentEnvVarArgument(lightrunJavaAgent.Spec.InitContainer.SharedVolumeMountPath, lightrunJavaAgent.Spec.AgentCliFlags)
	if err != nil {
		log.Error(err, "agentEnvVarArgument exceeds 1024 chars")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Create config map
	log.V(2).Info("Reconciling config map with agent configuration")
	configMap, err := r.createAgentConfig(lightrunJavaAgent)
	if err != nil {
		log.Error(err, "unable to create configMap")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("lightrun-controller")}

	err = r.Patch(ctx, &configMap, client.Apply, applyOpts...)
	if err != nil {
		log.Error(err, "unable to create configMap")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Calculate ConfigMap Data hash for deployment rollout trigger
	cmNamespacedObj := client.ObjectKey{
		Name:      configMap.Name,
		Namespace: configMap.Namespace,
	}
	cm := &corev1.ConfigMap{}
	err = r.Get(ctx, cmNamespacedObj, cm)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Error(err, "ConfigMap not found", "CM", configMap.Name)
		}
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}
	cmDataHash := configMapDataHash(cm.Data)

	// Server side apply
	log.V(2).Info("Patching deployment, SSA", "Deployment", deploymentName, "LightunrJavaAgent", lightrunJavaAgent.Name)
	err = r.patchDeployment(lightrunJavaAgent, secret, originalDeployment, deploymentApplyConfig, cmDataHash)
	if err != nil {
		log.Error(err, "unable to patch deployment")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deploymentApplyConfig)
	if err != nil {
		log.Error(err, "failed to convert Deployment to unstructured")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	patch := &unstructured.Unstructured{
		Object: obj,
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: fieldManager,
		Force:        pointer.Bool(true),
	})
	if err != nil {
		log.Error(err, "failed to patch deployment")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Client side patch (we can't rollback JAVA_TOOL_OPTIONS env with server side apply)
	log.V(2).Info("Patching Java Env", "Deployment", lightrunJavaAgent.Spec.DeploymentName, "LightunrJavaAgent", lightrunJavaAgent.Name)
	originalDeployment = &appsv1.Deployment{}
	err = r.Get(ctx, deplNamespacedObj, originalDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info("Deployment not found", "Deployment", lightrunJavaAgent.Spec.DeploymentName)
			err = r.removeFinalizer(ctx, lightrunJavaAgent, finalizerName)
			if err != nil {
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}
			return r.errorStatus(ctx, lightrunJavaAgent, errors.New("deployment not found"))
		}
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}
	clientSidePatch := client.MergeFrom(originalDeployment.DeepCopy())
	for i, container := range originalDeployment.Spec.Template.Spec.Containers {
		for _, targetContainer := range lightrunJavaAgent.Spec.ContainerSelector {
			if targetContainer == container.Name {
				err = r.patchJavaToolEnv(originalDeployment.Annotations, &originalDeployment.Spec.Template.Spec.Containers[i], lightrunJavaAgent.Spec.AgentEnvVarName, agentArg)
				if err != nil {
					log.Error(err, "failed to patch "+lightrunJavaAgent.Spec.AgentEnvVarName)
					return r.errorStatus(ctx, lightrunJavaAgent, err)
				}
			}
		}
	}
	originalDeployment.Annotations[annotationPatchedEnvName] = lightrunJavaAgent.Spec.AgentEnvVarName
	originalDeployment.Annotations[annotationPatchedEnvValue] = agentArg
	err = r.Patch(ctx, originalDeployment, clientSidePatch)
	if err != nil {
		log.Error(err, "failed to patch "+lightrunJavaAgent.Spec.AgentEnvVarName)
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Update status to Healthy
	log.V(1).Info("Reconciling finished successfully", "Deployment", deploymentName, "LightunrJavaAgent", lightrunJavaAgent.Name)
	return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeReady)
}

// reconcileStatefulSet handles the reconciliation logic for StatefulSet workloads
func (r *LightrunJavaAgentReconciler) reconcileStatefulSet(ctx context.Context, lightrunJavaAgent *agentv1beta.LightrunJavaAgent, namespace string) (ctrl.Result, error) {
	log := r.Log.WithValues("lightrunJavaAgent", lightrunJavaAgent.Name, "statefulSet", lightrunJavaAgent.Spec.WorkloadName)
	fieldManager := "lightrun-controller"

	stsNamespacedObj := client.ObjectKey{
		Name:      lightrunJavaAgent.Spec.WorkloadName,
		Namespace: namespace,
	}
	originalStatefulSet := &appsv1.StatefulSet{}
	err = r.Get(ctx, stsNamespacedObj, originalStatefulSet)
	if err != nil {
		// StatefulSet not found
		if client.IgnoreNotFound(err) == nil {
			log.Info("StatefulSet not found. Verify name/namespace", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName)
			// remove our finalizer from the list and update it.
			err = r.removeFinalizer(ctx, lightrunJavaAgent, finalizerName)
			if err != nil {
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}
			return r.errorStatus(ctx, lightrunJavaAgent, errors.New("statefulset not found"))
		} else {
			log.Error(err, "unable to fetch statefulset")
			return r.errorStatus(ctx, lightrunJavaAgent, err)
		}
	}

	// Check if this LightrunJavaAgent is being deleted
	if !lightrunJavaAgent.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if containsString(lightrunJavaAgent.ObjectMeta.Finalizers, finalizerName) {
			// our finalizer is present, so lets handle any cleanup operations

			// Restore original StatefulSet (unpatch)
			// Volume and init container
			log.Info("Unpatching StatefulSet", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName)

			originalStatefulSet = &appsv1.StatefulSet{}
			err = r.Get(ctx, stsNamespacedObj, originalStatefulSet)
			if err != nil {
				if client.IgnoreNotFound(err) == nil {
					log.Info("StatefulSet not found", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName)
					// remove our finalizer from the list and update it.
					log.Info("Removing finalizer")
					err = r.removeFinalizer(ctx, lightrunJavaAgent, finalizerName)
					if err != nil {
						return r.errorStatus(ctx, lightrunJavaAgent, err)
					}
					// Successfully removed finalizer and nothing to restore
					return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeReady)
				}
				log.Error(err, "unable to unpatch statefulset", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName)
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}

			// Revert environment variable modifications
			clientSidePatch := client.MergeFrom(originalStatefulSet.DeepCopy())
			for i, container := range originalStatefulSet.Spec.Template.Spec.Containers {
				for _, targetContainer := range lightrunJavaAgent.Spec.ContainerSelector {
					if targetContainer == container.Name {
						r.unpatchJavaToolEnv(originalStatefulSet.Annotations, &originalStatefulSet.Spec.Template.Spec.Containers[i])
					}
				}
			}
			delete(originalStatefulSet.Annotations, annotationPatchedEnvName)
			delete(originalStatefulSet.Annotations, annotationPatchedEnvValue)
			delete(originalStatefulSet.Annotations, annotationAgentName)
			err = r.Patch(ctx, originalStatefulSet, clientSidePatch)
			if err != nil {
				log.Error(err, "failed to unpatch statefulset environment variables")
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}

			// Remove Volumes and init container
			emptyApplyConfig := appsv1ac.StatefulSet(stsNamespacedObj.Name, stsNamespacedObj.Namespace)
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(emptyApplyConfig)
			if err != nil {
				log.Error(err, "failed to convert StatefulSet to unstructured")
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}
			patch := &unstructured.Unstructured{
				Object: obj,
			}
			err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
				FieldManager: fieldManager,
				Force:        pointer.Bool(true),
			})
			if err != nil {
				log.Error(err, "failed to unpatch statefulset")
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}

			// remove our finalizer from the list and update it.
			log.Info("Removing finalizer")
			err = r.removeFinalizer(ctx, lightrunJavaAgent, finalizerName)
			if err != nil {
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}

			log.Info("StatefulSet returned to original state", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName)
			return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeProgressing)
		}
		// Nothing to do here
		return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeProgressing)
	}

	// Check if already patched by another LightrunJavaAgent
	if oldLrjaName, ok := originalStatefulSet.Annotations[annotationAgentName]; ok && oldLrjaName != lightrunJavaAgent.Name {
		log.Error(err, "StatefulSet already patched by LightrunJavaAgent", "Existing LightrunJavaAgent", oldLrjaName)
		return r.errorStatus(ctx, lightrunJavaAgent, errors.New("statefulset already patched"))
	}

	// Add finalizer if not already present
	if !containsString(lightrunJavaAgent.ObjectMeta.Finalizers, finalizerName) {
		log.V(2).Info("Adding finalizer")
		err = r.addFinalizer(ctx, lightrunJavaAgent, finalizerName)
		if err != nil {
			log.Error(err, "unable to add finalizer")
			return r.errorStatus(ctx, lightrunJavaAgent, err)
		}
	}

	// Get the secret
	secretObj := client.ObjectKey{
		Name:      lightrunJavaAgent.Spec.SecretName,
		Namespace: namespace,
	}
	secret = &corev1.Secret{}
	err = r.Get(ctx, secretObj, secret)
	if err != nil {
		log.Error(err, "unable to fetch Secret", "Secret", lightrunJavaAgent.Spec.SecretName)
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Verify that env var won't exceed 1024 chars
	agentArg, err := agentEnvVarArgument(lightrunJavaAgent.Spec.InitContainer.SharedVolumeMountPath, lightrunJavaAgent.Spec.AgentCliFlags)
	if err != nil {
		log.Error(err, "agentEnvVarArgument exceeds 1024 chars")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Create config map
	log.V(2).Info("Reconciling config map with agent configuration")
	configMap, err := r.createAgentConfig(lightrunJavaAgent)
	if err != nil {
		log.Error(err, "unable to create configMap")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("lightrun-controller")}

	err = r.Patch(ctx, &configMap, client.Apply, applyOpts...)
	if err != nil {
		log.Error(err, "unable to apply configMap")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Calculate ConfigMap data hash
	cmDataHash := configMapDataHash(configMap.Data)

	// Extract StatefulSet for applying changes
	statefulSetApplyConfig, err := appsv1ac.ExtractStatefulSet(originalStatefulSet, fieldManager)
	if err != nil {
		log.Error(err, "failed to extract StatefulSet")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Server side apply for StatefulSet changes
	log.V(2).Info("Patching StatefulSet", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName, "LightunrJavaAgent", lightrunJavaAgent.Name)
	err = r.patchStatefulSet(lightrunJavaAgent, secret, originalStatefulSet, statefulSetApplyConfig, cmDataHash)
	if err != nil {
		log.Error(err, "failed to patch statefulset")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(statefulSetApplyConfig)
	if err != nil {
		log.Error(err, "failed to convert StatefulSet to unstructured")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}
	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: fieldManager,
		Force:        pointer.Bool(true),
	})
	if err != nil {
		log.Error(err, "failed to patch statefulset")
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Client side patch (we can't rollback JAVA_TOOL_OPTIONS env with server side apply)
	log.V(2).Info("Patching Java Env", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName, "LightunrJavaAgent", lightrunJavaAgent.Name)
	originalStatefulSet = &appsv1.StatefulSet{}
	err = r.Get(ctx, stsNamespacedObj, originalStatefulSet)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info("StatefulSet not found", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName)
			err = r.removeFinalizer(ctx, lightrunJavaAgent, finalizerName)
			if err != nil {
				return r.errorStatus(ctx, lightrunJavaAgent, err)
			}
			return r.errorStatus(ctx, lightrunJavaAgent, errors.New("statefulset not found"))
		}
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}
	clientSidePatch := client.MergeFrom(originalStatefulSet.DeepCopy())
	for i, container := range originalStatefulSet.Spec.Template.Spec.Containers {
		for _, targetContainer := range lightrunJavaAgent.Spec.ContainerSelector {
			if targetContainer == container.Name {
				err = r.patchJavaToolEnv(originalStatefulSet.Annotations, &originalStatefulSet.Spec.Template.Spec.Containers[i], lightrunJavaAgent.Spec.AgentEnvVarName, agentArg)
				if err != nil {
					log.Error(err, "failed to patch "+lightrunJavaAgent.Spec.AgentEnvVarName)
					return r.errorStatus(ctx, lightrunJavaAgent, err)
				}
			}
		}
	}
	originalStatefulSet.Annotations[annotationPatchedEnvName] = lightrunJavaAgent.Spec.AgentEnvVarName
	originalStatefulSet.Annotations[annotationPatchedEnvValue] = agentArg
	err = r.Patch(ctx, originalStatefulSet, clientSidePatch)
	if err != nil {
		log.Error(err, "failed to patch "+lightrunJavaAgent.Spec.AgentEnvVarName)
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Update status to Healthy
	log.V(1).Info("Reconciling finished successfully", "StatefulSet", lightrunJavaAgent.Spec.WorkloadName, "LightunrJavaAgent", lightrunJavaAgent.Name)
	return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeReady)
}

// SetupWithManager configures the controller with the Manager and sets up watches and indexers.
// It creates several field indexers to enable efficient lookups of LightrunJavaAgent CRs based on:
// - DeploymentName (legacy field)
// - WorkloadName (newer field that replaces DeploymentName)
// - SecretName
//
// It also sets up watches for Deployments, StatefulSets, and Secrets so the controller can
// react to changes in these resources that are referenced by LightrunJavaAgent CRs.
func (r *LightrunJavaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index field for deployments - allows looking up LightrunJavaAgents by deploymentName
	// This is used for legacy support where DeploymentName was used instead of WorkloadName
	// TODO: remove this once we deprecate deploymentNameIndexField
	err = mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentv1beta.LightrunJavaAgent{},
		deploymentNameIndexField,
		func(object client.Object) []string {
			agent := object.(*agentv1beta.LightrunJavaAgent)
			if agent.Spec.DeploymentName == "" {
				return nil
			}
			r.Log.Info("Indexing DeploymentName", "DeploymentName", agent.Spec.DeploymentName)
			return []string{agent.Spec.DeploymentName}
		})
	if err != nil {
		return err
	}

	// Index field for workloads by name - allows looking up LightrunJavaAgents by WorkloadName
	err = mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentv1beta.LightrunJavaAgent{},
		workloadNameIndexField,
		func(object client.Object) []string {
			agent := object.(*agentv1beta.LightrunJavaAgent)
			if agent.Spec.WorkloadName == "" {
				return nil
			}
			r.Log.Info("Indexing WorkloadName", "WorkloadName", agent.Spec.WorkloadName)
			return []string{agent.Spec.WorkloadName}
		})
	if err != nil {
		return err
	}

	// Index field for secrets - allows looking up LightrunJavaAgents by SecretName
	// This enables the controller to find LightrunJavaAgents affected by Secret changes
	err = mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentv1beta.LightrunJavaAgent{},
		secretNameIndexField,
		func(object client.Object) []string {
			lightrunJavaAgent := object.(*agentv1beta.LightrunJavaAgent)

			if lightrunJavaAgent.Spec.SecretName == "" {
				return nil
			}

			return []string{lightrunJavaAgent.Spec.SecretName}
		})

	if err != nil {
		return err
	}

	// Configure the controller builder:
	// - For: register LightrunJavaAgent as the primary resource this controller reconciles
	// - Watches: set up event handlers to watch for changes in related resources:
	//   * Deployments: reconcile LightrunJavaAgents when their target Deployment changes
	//   * StatefulSets: reconcile LightrunJavaAgents when their target StatefulSet changes
	//   * Secrets: reconcile LightrunJavaAgents when their referenced Secret changes
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentv1beta.LightrunJavaAgent{}).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.mapDeploymentToAgent),
		).
		Watches(
			&appsv1.StatefulSet{},
			handler.EnqueueRequestsFromMapFunc(r.mapStatefulSetToAgent),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToAgent),
		).
		Complete(r)
}
