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

package v1beta

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make" to regenerate code after modifying this file
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkloadType defines the type of workload that can be patched
// +kubebuilder:validation:Enum=Deployment;StatefulSet
type WorkloadType string

const (
	// WorkloadTypeDeployment represents a Kubernetes Deployment
	WorkloadTypeDeployment WorkloadType = "Deployment"
	// WorkloadTypeStatefulSet represents a Kubernetes StatefulSet
	WorkloadTypeStatefulSet WorkloadType = "StatefulSet"
)

type InitContainer struct {
	// Name of the volume that will be added to pod
	SharedVolumeName string `json:"sharedVolumeName"`
	// Path in the app container where volume with agent will be mounted
	SharedVolumeMountPath string `json:"sharedVolumeMountPath"`
	// Image of the init container. Image name and tag will define platform and version of the agent
	Image string `json:"image"`
	// Pull policy of the init container. Can be one of: Always, IfNotPresent, or Never.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// LightrunJavaAgentSpec defines the desired state of LightrunJavaAgent
type LightrunJavaAgentSpec struct {
	// List of containers that should be patched in the Pod
	ContainerSelector []string      `json:"containerSelector"`
	InitContainer     InitContainer `json:"initContainer"`

	// Name of the Deployment that will be patched. Deprecated, use WorkloadName and WorkloadType instead
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// Name of the Workload that will be patched. workload can be either Deployment or StatefulSet e.g. my-deployment, my-statefulset
	// +optional
	WorkloadName string `json:"workloadName,omitempty"`

	// Type of the workload that will be patched supported values are Deployment, StatefulSet
	// +optional
	WorkloadType WorkloadType `json:"workloadType,omitempty"`

	//Name of the Secret in the same namespace contains lightrun key and conmpany id
	SecretName string `json:"secretName"`

	//Env variable that will be patched with the -agentpath
	//Common choice is JAVA_TOOL_OPTIONS
	//Depending on the tool used it may vary from JAVA_OPTS to MAVEN_OPTS and CATALINA_OPTS
	// More info can be found here https://docs.lightrun.com/jvm/build-tools/
	AgentEnvVarName string `json:"agentEnvVarName"`

	// Lightrun server hostname that will be used for downloading an agent
	// Key and company id in the secret has to be taken from this server as well
	ServerHostname string `json:"serverHostname"`

	// Agent configuration to be changed from default values
	// https://docs.lightrun.com/jvm/agent-configuration/#setting-agent-properties-from-the-agentconfig-file
	// +optional
	AgentConfig map[string]string `json:"agentConfig,omitempty"`

	// Add cli flags to the agent "-agentpath:/lightrun/agent/lightrun_agent.so=<AgentCliFlags>"
	// https://docs.lightrun.com/jvm/agent-configuration/#additional-command-line-flags
	// +optional
	AgentCliFlags string `json:"agentCliFlags,omitempty"`

	// Agent tags that will be shown in the portal / IDE plugin
	AgentTags []string `json:"agentTags"`

	// +optional
	// Agent name for registration to the server
	AgentName string `json:"agentName,omitempty"`
}

// LightrunJavaAgentStatus defines the observed state of LightrunJavaAgent
type LightrunJavaAgentStatus struct {
	LastScheduleTime *metav1.Time       `json:"lastScheduleTime,omitempty"`
	Conditions       []metav1.Condition `json:"conditions,omitempty"`
	WorkloadStatus   string             `json:"workloadStatus,omitempty"`
	DeploymentStatus string             `json:"deploymentStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=lrja
//+kubebuilder:printcolumn:priority=0,name=Workload,type=string,JSONPath=".spec.workloadName",description="Workload name",format=""
//+kubebuilder:printcolumn:priority=0,name=Type,type=string,JSONPath=".spec.workloadType",description="Workload type",format=""
//+kubebuilder:printcolumn:priority=0,name="Status",type=string,JSONPath=".status.workloadStatus",description="Status of Workload Reconciliation",format=""
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LightrunJavaAgent is the Schema for the lightrunjavaagents API
type LightrunJavaAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LightrunJavaAgentSpec   `json:"spec,omitempty"`
	Status LightrunJavaAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// LightrunJavaAgentList contains a list of LightrunJavaAgent
type LightrunJavaAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LightrunJavaAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LightrunJavaAgent{}, &LightrunJavaAgentList{})
}
