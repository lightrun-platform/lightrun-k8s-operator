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
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;watch;list

func (r *LightrunJavaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("lightrunJavaAgent", req.NamespacedName)
	fieldManager := "lightrun-conrtoller"
	lightrunJavaAgent := &agentv1beta.LightrunJavaAgent{}
	if err = r.Get(ctx, req.NamespacedName, lightrunJavaAgent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deplNamespacedObj := client.ObjectKey{
		Name:      lightrunJavaAgent.Spec.DeploymentName,
		Namespace: req.Namespace,
	}
	originalDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, deplNamespacedObj, originalDeployment)
	if err != nil {
		// Deployment not found
		if client.IgnoreNotFound(err) == nil {
			log.Info("Deployment not found. Verify name/namespace", "Deployment", lightrunJavaAgent.Spec.DeploymentName)
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

	if oldLrjaName, ok := originalDeployment.Annotations["lightrun.com/lightrunjavaagent"]; ok && oldLrjaName != lightrunJavaAgent.Name {
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
			Namespace: req.Namespace,
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
			log.Info("Start working on deployment", "Deployment", lightrunJavaAgent.Spec.DeploymentName)
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
			delete(originalDeployment.Annotations, "lightrun.com/patched-env-name")
			delete(originalDeployment.Annotations, "lightrun.com/patched-env-value")
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

			log.Info("Deployment returned to original state", "Deployment", lightrunJavaAgent.Spec.DeploymentName)
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
	cmDataHash := hash(cm.Data["config"] + cm.Data["metadata"])

	// Server side apply
	log.V(2).Info("Patching deployment, SSA", "Deployment", lightrunJavaAgent.Spec.DeploymentName, "LightunrJavaAgent", lightrunJavaAgent.Name)
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
	originalDeployment.Annotations["lightrun.com/patched-env-name"] = lightrunJavaAgent.Spec.AgentEnvVarName
	originalDeployment.Annotations["lightrun.com/patched-env-value"] = agentArg
	err = r.Patch(ctx, originalDeployment, clientSidePatch)
	if err != nil {
		log.Error(err, "failed to patch "+lightrunJavaAgent.Spec.AgentEnvVarName)
		return r.errorStatus(ctx, lightrunJavaAgent, err)
	}

	// Update status to Healthy
	log.V(1).Info("Reconciling finished successfully", "Deployment", lightrunJavaAgent.Spec.DeploymentName, "LightunrJavaAgent", lightrunJavaAgent.Name)
	return r.successStatus(ctx, lightrunJavaAgent, reconcileTypeReady)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LightrunJavaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add spec.container_selector.deployment field to cache for future filtering
	err = mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentv1beta.LightrunJavaAgent{},
		deploymentNameIndexField,
		func(object client.Object) []string {
			lightrunJavaAgent := object.(*agentv1beta.LightrunJavaAgent)

			if lightrunJavaAgent.Spec.DeploymentName == "" {
				return nil
			}

			return []string{lightrunJavaAgent.Spec.DeploymentName}
		})

	if err != nil {
		return err
	}

	// Add spec.container_selector.secret field to cache for future filtering
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&agentv1beta.LightrunJavaAgent{}).
		Owns(&corev1.ConfigMap{}).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.mapDeploymentToAgent),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToAgent),
		).
		Complete(r)
}
