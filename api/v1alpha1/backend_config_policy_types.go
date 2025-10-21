package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=backendconfigpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=backendconfigpolicies/status,verbs=get;update;patch

// +kubebuilder:printcolumn:name="Accepted",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Accepted')].status",description="Backend config policy acceptance status"
// +kubebuilder:printcolumn:name="Attached",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Attached')].status",description="Backend config policy attachment status"

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels={app=kgateway,app.kubernetes.io/name=kgateway,gateway.networking.k8s.io/policy=Direct}
// +kubebuilder:resource:categories=kgateway
// +kubebuilder:subresource:status
type BackendConfigPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackendPolicySpec `json:"spec,omitempty"`
	Status            gwv1.PolicyStatus       `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type BackendConfigPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackendConfigPolicy `json:"items"`
}

type BackendAuth struct {
	// Provide the token directly in the configuration for the Backend.
	// This option is the least secure. Only use this option for quick tests such as trying out AI Gateway.
	Inline *string `json:"inline,omitempty"`

	// Store the API key in a Kubernetes secret in the same namespace as the Backend.
	// Then, refer to the secret in the Backend configuration. This option is more secure than an inline token,
	// because the API key is encoded and you can restrict access to secrets through RBAC rules.
	// You might use this option in proofs of concept, controlled development and staging environments,
	// or well-controlled prod environments that use secrets.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// TODO: passthrough, aws, azure, gcp
}

type BackendAI struct {
	// Enrich requests sent to the LLM provider by appending and prepending system prompts.
	// This can be configured only for LLM providers that use the `CHAT` or `CHAT_STREAMING` API route type.
	PromptEnrichment *AIPromptEnrichment `json:"prompt,omitempty"`

	// TODO: the API here is very messy and confusing; do a general refactoring
	PromptGuard *AIPromptGuard `json:"promptGuard,omitempty"`

	// Provide defaults to merge with user input fields.
	Defaults []FieldDefault `json:"defaults,omitempty"`
	Overrides []FieldDefault `json:"overrides,omitempty"`
	// Intentionally omitted: `model`. Instead, use overrides.

	// ModelAliases maps friendly model names to actual provider model names.
	// Example: {"fast": "gpt-3.5-turbo", "smart": "gpt-4-turbo"}
	// Note: This field is only applicable when using the agentgateway data plane.
	// TODO: should this use 'overrides', and we add CEL conditionals?
	// +optional
	ModelAliases map[string]string `json:"modelAliases,omitempty"`
}

type BackendMCP struct {
	Authorization *Authorization `json:"authorization,omitempty"`
	Authentication *MCPAuthentication `json:"authentication,omitempty"`
}
type MCPAuthentication struct {

}
type BackendHTTP struct {
	IdleTimeout *metav1.Duration `json:"idleTimeout,omitempty"`

	InitialStreamWindowSize     *resource.Quantity `json:"initialStreamWindowSize,omitempty"`
	InitialConnectionWindowSize *resource.Quantity `json:"initialConnectionWindowSize,omitempty"`
	MaxConcurrentStreams *int32 `json:"maxConcurrentStreams,omitempty"`
}
type BackendTCP struct {
	// Configure OS-level TCP keepalive checks.
	// +optional
	Keepalive *TCPKeepalive `json:"keepalive,omitempty"`
	// The timeout for new network connections to hosts in the cluster.
	// +optional
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	ConnectTimeout *metav1.Duration `json:"connectTimeout,omitempty"`
}

// BackendConfigPolicySpec defines the desired state of BackendConfigPolicy.
//
// +kubebuilder:validation:AtMostOneOf=http1ProtocolOptions;http2ProtocolOptions
type BackendPolicySpec struct {
	// TargetRefs specifies the target references to attach the policy to.
	//
	// While a BackendPolicy applies to backend resources (Service or Backend), to apply a configuration to group of backends
	// a BackendPolicy can target a higher level resource like a Gateway, to apply defaults for all backends used by that Gateway.
	//
	// When multiple policies are applied, the following order of precedence applies:
	// Gateway < Listener < Route < Route Rule < Service/Backend < Sub-backend
	//
	// +optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:XValidation:rule="self.all(r, (r.group == '' && r.kind == 'Service') || (r.group == 'gateway.kgateway.dev' && r.kind == 'Backend'))",message="TargetRefs must reference either a Kubernetes Service or a Backend API"
	TargetRefs []LocalPolicyTargetReference `json:"targetRefs,omitempty"`

	// TargetSelectors specifies the target selectors to select resources to attach the policy to.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.all(r, (r.group == '' && r.kind == 'Service') || (r.group == 'gateway.kgateway.dev' && r.kind == 'Backend'))",message="TargetSelectors must reference either a Kubernetes Service or a Backend API"
	TargetSelectors []LocalPolicyTargetSelector `json:"targetSelectors,omitempty"`

	TCP *BackendTCP `json:"tcp,omitempty"`

	HTTP *BackendHTTP `json:"http,omitempty"`

	MCP *BackendMCP `json:"mcp,omitempty"`
	AI *BackendAI `json:"ai,omitempty"`
	Auth *BackendAuth `json:"auth,omitempty"`

	// TLS contains the options necessary to configure a backend to use TLS origination.
	// See [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/transport_sockets/tls/v3/tls.proto#envoy-v3-api-msg-extensions-transport-sockets-tls-v3-sslconfig) for more details.
	// +optional
	TLS *TLS `json:"tls,omitempty"`
}

// See [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/address.proto#envoy-v3-api-msg-config-core-v3-tcpkeepalive) for more details.
type TCPKeepalive struct {
	// Maximum number of keep-alive probes to send before dropping the connection.
	// +optional
	// +kubebuilder:validation:Minimum=0
	KeepAliveProbes *int32 `json:"keepAliveProbes,omitempty"`

	// The number of seconds a connection needs to be idle before keep-alive probes start being sent.
	// +optional
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	// +kubebuilder:validation:XValidation:rule="duration(self) >= duration('1s')",message="keepAliveTime must be at least 1 second"
	KeepAliveTime *metav1.Duration `json:"keepAliveTime,omitempty"`

	// The number of seconds between keep-alive probes.
	// +optional
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	// +kubebuilder:validation:XValidation:rule="duration(self) >= duration('1s')",message="keepAliveInterval must be at least 1 second"
	KeepAliveInterval *metav1.Duration `json:"keepAliveInterval,omitempty"`
}

// +kubebuilder:validation:ExactlyOneOf=secretRef;files;insecureSkipVerify;wellKnownCACertificates
type TLS struct {
	// Reference to the TLS secret containing the certificate, key, and optionally the root CA.
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// InsecureSkipVerify originates TLS but skips verification of the backend's certificate.
	// WARNING: This is an insecure option that should only be used if the risks are understood.
	// +optional
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`

	// The SNI domains that should be used for TLS connection.
	// If unset, the destination's hostname will be used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Sni *string `json:"sni,omitempty"`

	// Verify that the Subject Alternative Name in the peer certificate is one of the specified values.
	// note that a root_ca must be provided if this option is used.
	// If unset, the SNI will be used.
	// +optional
	VerifySubjectAltNames []string `json:"verifySubjectAltNames,omitempty"`

	// General TLS parameters.
	// for more information on the meaning of these values.
	// +optional
	Parameters *TLSParameters `json:"parameters,omitempty"`

	// Set Application Level Protocol Negotiation
	// If empty, defaults to ["h2", "http/1.1"].
	// +optional
	AlpnProtocols []string `json:"alpnProtocols,omitempty"`
}

// TLSVersion defines the TLS version.
// +kubebuilder:validation:Enum=AUTO;"1.2";"1.3"
type TLSVersion string

const (
	TLSVersionAUTO TLSVersion = "AUTO"
	TLSVersion1_2  TLSVersion = "1.2"
	TLSVersion1_3  TLSVersion = "1.3"
)

type TLSParameters struct {
	// Minimum TLS version.
	// +optional
	MinVersion *TLSVersion `json:"minVersion,omitempty"`

	// Maximum TLS version.
	// +optional
	MaxVersion *TLSVersion `json:"maxVersion,omitempty"`

	// +optional
	CipherSuites []string `json:"cipherSuites,omitempty"`

	// +optional
	EcdhCurves []string `json:"ecdhCurves,omitempty"`
}
