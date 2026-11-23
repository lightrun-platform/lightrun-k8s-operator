package controller

import (
	"context"
	"encoding/json"
	"time"

	agentsv1beta "github.com/lightrun-platform/lightrun-k8s-operator/api/v1beta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	appsv1ac "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imgTimeout    = 20 * time.Second
	imgInterval   = 250 * time.Millisecond
	imgSecretName = "image-ownership-secret"
	imgAppImage   = "busybox:1.36"
)

func imgPodTemplate(name string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: imgAppImage}},
		},
	}
}

func imgAgent(name, workload string, wt agentsv1beta.WorkloadType) *agentsv1beta.LightrunJavaAgent {
	return &agentsv1beta.LightrunJavaAgent{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec: agentsv1beta.LightrunJavaAgentSpec{
			WorkloadName:      workload,
			WorkloadType:      wt,
			SecretName:        imgSecretName,
			ServerHostname:    "example.lightrun.com",
			AgentName:         "image-ownership-agent",
			AgentTags:         []string{"image-ownership"},
			AgentEnvVarName:   "JAVA_TOOL_OPTIONS",
			ContainerSelector: []string{"app"},
			InitContainer: agentsv1beta.InitContainer{
				Image:                 "lightruncom/lightrun-init-agent:latest",
				SharedVolumeName:      "lightrun-agent-init",
				SharedVolumeMountPath: "/lightrun",
			},
		},
	}
}

func imgAppContainerFields(managedFields []metav1.ManagedFieldsEntry, manager string) map[string]interface{} {
	for _, mf := range managedFields {
		if mf.Manager != manager || mf.Operation != metav1.ManagedFieldsOperationApply || mf.FieldsV1 == nil {
			continue
		}
		var fields map[string]interface{}
		Expect(json.Unmarshal(mf.FieldsV1.Raw, &fields)).To(Succeed())
		app, found, err := unstructured.NestedMap(fields, "f:spec", "f:template", "f:spec", "f:containers", `k:{"name":"app"}`)
		Expect(err).NotTo(HaveOccurred())
		if found {
			return app
		}
	}
	return nil
}

func imgIsPatched(spec corev1.PodSpec) bool {
	for _, c := range spec.InitContainers {
		if c.Name == initContainerName {
			return true
		}
	}
	return false
}

var _ = Describe("Application container image ownership", func() {
	ctx := context.Background()

	BeforeEach(func() {
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, &ns))).To(Succeed())
		s := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: imgSecretName, Namespace: testNamespace},
			StringData: map[string]string{"lightrun_key": "k", "pinned_cert_hash": "h"},
		}
		Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, &s))).To(Succeed())
	})

	It("does not take ownership of the Deployment container image", func() {
		key := types.NamespacedName{Name: "img-depl", Namespace: testNamespace}
		Expect(k8sClient.Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": key.Name}},
				Template: imgPodTemplate(key.Name),
			},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, imgAgent("img-depl-agent", key.Name, agentsv1beta.WorkloadTypeDeployment))).To(Succeed())

		var d appsv1.Deployment
		Eventually(func() bool {
			return k8sClient.Get(ctx, key, &d) == nil && imgIsPatched(d.Spec.Template.Spec)
		}, imgTimeout, imgInterval).Should(BeTrue())

		app := imgAppContainerFields(d.ManagedFields, "lightrun-conrtoller")
		Expect(app).To(HaveKey("f:volumeMounts"))
		Expect(app).NotTo(HaveKey("f:image"))
		Expect(d.Spec.Template.Spec.Containers[0].Image).To(Equal(imgAppImage))
	})

	It("does not take ownership of the StatefulSet container image", func() {
		key := types.NamespacedName{Name: "img-sts", Namespace: testNamespace}
		Expect(k8sClient.Create(ctx, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: key.Name,
				Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": key.Name}},
				Template:    imgPodTemplate(key.Name),
			},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, imgAgent("img-sts-agent", key.Name, agentsv1beta.WorkloadTypeStatefulSet))).To(Succeed())

		var s appsv1.StatefulSet
		Eventually(func() bool {
			return k8sClient.Get(ctx, key, &s) == nil && imgIsPatched(s.Spec.Template.Spec)
		}, imgTimeout, imgInterval).Should(BeTrue())

		app := imgAppContainerFields(s.ManagedFields, "lightrun-controller")
		Expect(app).To(HaveKey("f:volumeMounts"))
		Expect(app).NotTo(HaveKey("f:image"))
		Expect(s.Spec.Template.Spec.Containers[0].Image).To(Equal(imgAppImage))
	})

	It("releases the image owned by an older operator version and keeps the value", func() {
		key := types.NamespacedName{Name: "img-legacy", Namespace: testNamespace}
		Expect(k8sClient.Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": key.Name}},
				Template: imgPodTemplate(key.Name),
			},
		})).To(Succeed())

		legacy := appsv1ac.Deployment(key.Name, key.Namespace).WithSpec(appsv1ac.DeploymentSpec().WithTemplate(
			corev1ac.PodTemplateSpec().WithSpec(corev1ac.PodSpec().WithContainers(
				corev1ac.Container().WithName("app").WithImage(imgAppImage)))))
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(legacy)
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient.Patch(ctx, &unstructured.Unstructured{Object: obj}, client.Apply,
			client.FieldOwner("lightrun-conrtoller"), client.ForceOwnership)).To(Succeed())

		var d appsv1.Deployment
		Expect(k8sClient.Get(ctx, key, &d)).To(Succeed())
		Expect(imgAppContainerFields(d.ManagedFields, "lightrun-conrtoller")).To(HaveKey("f:image"))

		Expect(k8sClient.Create(ctx, imgAgent("img-legacy-agent", key.Name, agentsv1beta.WorkloadTypeDeployment))).To(Succeed())
		Eventually(func() bool {
			return k8sClient.Get(ctx, key, &d) == nil && imgIsPatched(d.Spec.Template.Spec)
		}, imgTimeout, imgInterval).Should(BeTrue())

		Expect(imgAppContainerFields(d.ManagedFields, "lightrun-conrtoller")).NotTo(HaveKey("f:image"))
		Expect(d.Spec.Template.Spec.Containers[0].Image).To(Equal(imgAppImage))

		Expect(k8sClient.Delete(ctx, imgAgent("img-legacy-agent", key.Name, agentsv1beta.WorkloadTypeDeployment))).To(Succeed())
		Eventually(func() bool {
			return apierrors.IsNotFound(k8sClient.Get(ctx, types.NamespacedName{Name: "img-legacy-agent", Namespace: testNamespace}, &agentsv1beta.LightrunJavaAgent{}))
		}, imgTimeout, imgInterval).Should(BeTrue())
		Expect(k8sClient.Get(ctx, key, &d)).To(Succeed())
		Expect(d.Spec.Template.Spec.Containers[0].Image).To(Equal(imgAppImage))
	})
})
