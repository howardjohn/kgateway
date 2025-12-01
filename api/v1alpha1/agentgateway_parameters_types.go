package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=agentgatewayparameters,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=agentgatewayparameters/status,verbs=get;update;patch

// +kubebuilder:printcolumn:name="Accepted",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Accepted')].status",description="Agentgateway policy acceptance status"
// +kubebuilder:printcolumn:name="Attached",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Attached')].status",description="Agentgateway policy attachment status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels={app=kgateway,app.kubernetes.io/name=kgateway}
// +kubebuilder:resource:categories=kgateway,shortName=agpar
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="gateway.networking.k8s.io/policy=Direct"
type AgentgatewayParameters struct {
	metav1.TypeMeta `json:",inline"`
	// metadata for the object
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of AgentgatewayParameters.
	// +required
	Spec AgentgatewayParametersSpec `json:"spec"`

	// status defines the current state of AgentgatewayParameters.
	// +optional
	Status AgentgatewayParametersStatus `json:"status,omitempty"`
}

// The current conditions of the GatewayParameters. This is not currently implemented.
type AgentgatewayParametersStatus struct{}

// +kubebuilder:object:root=true
type AgentgatewayParametersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentgatewayParameters `json:"items"`
}

type AgentgatewayParametersSpec struct {
	AgentgatewayParametersConfigs  `json:",inline"`
	AgentgatewayParametersOverlays `json:",inline"`
}

// +kubebuilder:validation:Enum=Json;Plain
type AgentgatewayParametersLoggingFormat string

const (
	AgentgatewayParametersLoggingJson  AgentgatewayParametersLoggingFormat = "Json"
	AgentgatewayParametersLoggingPlain AgentgatewayParametersLoggingFormat = "Plain"
)

type AgentgatewayParametersLogging struct {
	Level  ListOrString                        `json:"level,omitempty"`
	Format AgentgatewayParametersLoggingFormat `json:"format,omitempty"`
}

type AgentgatewayParametersConfigs struct {
	// Common set of labels to apply to all generated resources.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Common set of annotations to apply to all generated resources.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// logging configuration for Agentgateway. By default, all logs are set to "info" level.
	// +optional
	Logging *AgentgatewayParametersLogging `json:"logging,omitempty"`
	// The agentgateway container image. See
	// https://kubernetes.io/docs/concepts/containers/images
	// for details.
	//
	// Default values, which may be overridden individually:
	//
	//	registry: ghcr.io/agentgateway
	//	repository: agentgateway
	//	tag: <agentgateway version>
	//	pullPolicy: IfNotPresent
	//
	// +optional
	Image *Image `json:"image,omitempty"`
	// The container environment variables.
	//
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// The compute resources required by this container. See
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// for details.
	//
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type AgentgatewayParametersOverlays struct {
	// deployment allows specifying overrides for the generated Deployment resource.
	Deployment *AgentgatewayParametersObjectOverlay `json:"deployment,omitempty"`
	// service allows specifying overrides for the generated Service resource.
	Service *AgentgatewayParametersObjectOverlay `json:"service,omitempty"`
	// serviceAccount allows specifying overrides for the generated ServiceAccount resource.
	ServiceAccount *AgentgatewayParametersObjectOverlay `json:"serviceAccount,omitempty"`
	// podDisruptionBudget allows specifying overrides for the generated PodDisruptionBudget resource.
	// Note: a PodDisruptionBudget is not deployed by default. Setting this field enables a default one.
	// If you just want the default, without customizations, use `podDisruptionBudget: {}`.
	PodDisruptionBudget *AgentgatewayParametersObjectOverlay `json:"podDisruptionBudget,omitempty"`
	// horizontalPodAutoscaler allows specifying overrides for the generated HorizontalPodAutoscaler resource.
	// Note: a HorizontalPodAutoscaler is not deployed by default. Setting this field enables a default one.
	// If you just want the default, without customizations, use `horizontalPodAutoscaler: {}`.
	HorizontalPodAutoscaler *AgentgatewayParametersObjectOverlay `json:"horizontalPodAutoscaler,omitempty"`
}

type AgentgatewayParametersObjectMetadata struct {
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}
type AgentgatewayParametersObjectOverlay struct {
	// metadata defines a subset of object metadata to be customized.
	// +optional
	Metadata AgentgatewayParametersObjectMetadata `json:"metadata,omitempty"`
	// spec defines an overlay to apply onto the object, using [Strategic Merge Patch](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md).
	// The patch is applied after all other fields are applied.
	// +optional
	Spec apiextensionsv1.JSON `json:"spec,omitempty"`
}

// TODO: this doesn't work
// ListOrString is a type that can hold either a single string or a list of strings
// +kubebuilder:validation:Type=array
// +kubebuilder:validation:Type=string
type ListOrString []string

// UnmarshalJSON implements the json.Unmarshaller interface
func (l *ListOrString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	// Try to unmarshal as string first
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		*l = strings.Split(strVal, ",")
		return nil
	}

	// Try to unmarshal as array
	var arrVal []string
	if err := json.Unmarshal(data, &arrVal); err == nil {
		*l = arrVal
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into ListOrString", string(data))
}
