package controller

import (
	"context"
	"errors"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	agentv1beta "github.com/lightrun-platform/lightrun-k8s-operator/api/v1beta"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	reconcileTypeReady          = "Ready"
	reconcileTypeProgressing    = "ReconcileProgressing"
	reconcileTypeNotProgressing = "ReconcileFailed"
)

func (r *LightrunJavaAgentReconciler) mapDeploymentToAgent(ctx context.Context, obj client.Object) []reconcile.Request {
	deployment := obj.(*appsv1.Deployment)

	var lightrunJavaAgentList agentv1beta.LightrunJavaAgentList

	if err := r.List(ctx, &lightrunJavaAgentList,
		client.InNamespace(deployment.Namespace),
		client.MatchingFields{deploymentNameIndexField: deployment.Name},
	); err != nil {
		r.Log.Error(err, "could not list LightrunJavaAgentList. "+
			"change to deployment will not be reconciled.",
			deployment.Name, deployment.Namespace)
		return nil
	}

	requests := make([]reconcile.Request, len(lightrunJavaAgentList.Items))

	for i, lightrunJavaAgent := range lightrunJavaAgentList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&lightrunJavaAgent),
		}
	}
	return requests
}

func (r *LightrunJavaAgentReconciler) mapSecretToAgent(ctx context.Context, obj client.Object) []reconcile.Request {
	secret := obj.(*corev1.Secret)

	var lightrunJavaAgentList agentv1beta.LightrunJavaAgentList

	if err := r.List(ctx, &lightrunJavaAgentList,
		client.InNamespace(secret.Namespace),
		client.MatchingFields{secretNameIndexField: secret.Name},
	); err != nil {
		r.Log.Error(err, "could not list LightrunJavaAgentList. "+
			"change to secret will not be reconciled.",
			secret.Name, secret.Namespace)
		return nil
	}

	requests := make([]reconcile.Request, len(lightrunJavaAgentList.Items))

	for i, lightrunJavaAgent := range lightrunJavaAgentList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&lightrunJavaAgent),
		}
	}
	return requests
}

func (r *LightrunJavaAgentReconciler) addFinalizer(ctx context.Context, lightrunJavaAgent *agentv1beta.LightrunJavaAgent, finalizerName string) error {
	patch := client.MergeFrom(lightrunJavaAgent.DeepCopy())
	lightrunJavaAgent.ObjectMeta.Finalizers = append(lightrunJavaAgent.ObjectMeta.Finalizers, finalizerName)
	err := r.Patch(ctx, lightrunJavaAgent, patch)
	return err
}

func (r *LightrunJavaAgentReconciler) removeFinalizer(ctx context.Context, lightrunJavaAgent *agentv1beta.LightrunJavaAgent, finalizerName string) error {
	patch := client.MergeFrom(lightrunJavaAgent.DeepCopy())
	lightrunJavaAgent.ObjectMeta.Finalizers = removeString(lightrunJavaAgent.ObjectMeta.Finalizers, finalizerName)
	err := r.Patch(ctx, lightrunJavaAgent, patch)
	return err
}

func (r *LightrunJavaAgentReconciler) successStatus(ctx context.Context, instance *agentv1beta.LightrunJavaAgent, reconcileType string) (reconcile.Result, error) {
	condition := metav1.Condition{
		Type:               reconcileType,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: instance.GetGeneration(),
		Reason:             "reconcileSucceeded",
		Status:             metav1.ConditionTrue,
	}
	SetStatusCondition(&instance.Status.Conditions, condition)
	instance.Status.DeploymentStatus = r.findLastConditionType(&instance.Status.Conditions)
	err := r.Status().Update(ctx, instance)
	if err != nil {
		if apierrors.IsConflict(err) {
			r.Log.V(2).Info("unable to update status for", "object version", instance.GetResourceVersion(), "resource version expired, will trigger another reconcile cycle", "")
			return reconcile.Result{Requeue: true}, nil
		} else {
			r.Log.Error(err, "unable to update status for", "object", instance)
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *LightrunJavaAgentReconciler) errorStatus(ctx context.Context, instance *agentv1beta.LightrunJavaAgent, origError error) (reconcile.Result, error) {

	condition := metav1.Condition{
		Type:               reconcileTypeNotProgressing,
		LastTransitionTime: metav1.Now(),
		Message:            origError.Error(),
		ObservedGeneration: instance.GetGeneration(),
		Reason:             "reconcileFailed",
		Status:             metav1.ConditionTrue,
	}
	SetStatusCondition(&instance.Status.Conditions, condition)
	instance.Status.DeploymentStatus = r.findLastConditionType(&instance.Status.Conditions)
	err := r.Status().Update(ctx, instance)
	if err != nil {
		if apierrors.IsConflict(err) {
			r.Log.Info("unable to update status for", "object version", instance.GetResourceVersion(), "resource version expired, will trigger another reconcile cycle", "")
			return reconcile.Result{Requeue: true}, nil
		} else {
			r.Log.Error(err, "unable to update status for", "object", instance)
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, origError
}

func (r *LightrunJavaAgentReconciler) findLastConditionType(conditions *[]metav1.Condition) string {
	index := -1
	var ts metav1.Time
	for i, cond := range *conditions {
		if index == -1 {
			index = i
			ts = cond.LastTransitionTime
			continue
		}
		if ts.Before(&cond.LastTransitionTime) {
			ts = cond.LastTransitionTime
			index = i
		}
	}
	return (*conditions)[index].Type
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func parseAgentConfig(config map[string]string) string {
	var cm_data string
	var keys []string
	// sort keys to preserve order in hash
	for k := range config {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		cm_data = cm_data + k + "=" + config[k] + "\n"
	}
	return cm_data
}

func populateTags(tags []string, name string, metadata *AgentMetadata) {
	newTags := []Tag{}
	for _, tag := range tags {
		newTags = append(newTags, Tag{Name: tag})
	}
	metadata.Registration.DisplayName = name
	metadata.Registration.Tags = newTags
}

func hash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func SetStatusCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	if conditions == nil {
		return
	}
	existingCondition := meta.FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		if newCondition.LastTransitionTime.IsZero() {
			newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}
		*conditions = append(*conditions, newCondition)
		return
	}

	existingCondition.Status = newCondition.Status
	if !newCondition.LastTransitionTime.IsZero() {
		existingCondition.LastTransitionTime = newCondition.LastTransitionTime
	} else {
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	existingCondition.ObservedGeneration = newCondition.ObservedGeneration
}

func agentEnvVarArgument(mountPath string, agentCliFlags string) (string, error) {
	agentArg := " -agentpath:" + mountPath + "/agent/lightrun_agent.so"
	if agentCliFlags != "" {
		agentArg += "=" + agentCliFlags
		if len(agentArg) > 1024 {
			return "", errors.New("agentpath with agentCliFlags has more than 1024 chars. This is a limitation of Java")
		}
	}
	return agentArg, nil
}

// Removes from env var value. Removes env var from the list if value is empty after the update
func unpatchEnvVarValue(origValue string, removalValue string) string {
	value := strings.ReplaceAll(origValue, removalValue, "")
	return value
}

// Return index if the env var in the []corev1.EnvVar, otherwise -1
func findEnvVarIndex(envVarName string, envVarList []corev1.EnvVar) int {
	for i, envVar := range envVarList {
		if envVar.Name == envVarName {
			return i
		}
	}
	return -1
}
