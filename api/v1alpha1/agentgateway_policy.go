package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=agentgatwaypolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=agentgatwaypolicies/status,verbs=get;update;patch

// +kubebuilder:printcolumn:name="Accepted",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Accepted')].status",description="Agentgateway policy acceptance status"
// +kubebuilder:printcolumn:name="Attached",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Attached')].status",description="Agentgateway policy attachment status"

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels={app=kgateway,app.kubernetes.io/name=kgateway}
// +kubebuilder:resource:categories=kgateway
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="gateway.networking.k8s.io/policy=Direct"
type AgentgatewayPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AgentgatewayPolicySpec `json:"spec,omitempty"`

	Status gwv1.PolicyStatus `json:"status,omitempty"`
	// TODO: embed this into a typed Status field when
	// https://github.com/kubernetes/kubernetes/issues/131533 is resolved
}
type AgentgatewayPolicySpec struct {
	// TargetRefs specifies the target resources by reference to attach the policy to.
	// +optional
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:XValidation:rule="self.all(r, (r.kind == 'Backend' || r.kind == 'Gateway' || r.kind == 'HTTPRoute' || (r.kind == 'XListenerSet' && r.group == 'gateway.networking.x-k8s.io')) && (!has(r.group) || r.group == 'gateway.networking.k8s.io' || r.group == 'gateway.networking.x-k8s.io' || r.group == 'gateway.kgateway.dev' ))",message="targetRefs may only reference Gateway, HTTPRoute, XListenerSet, or Backend resources"
	TargetRefs []LocalPolicyTargetReferenceWithSectionName `json:"targetRefs,omitempty"`

	// TargetSelectors specifies the target selectors to select resources to attach the policy to.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.all(r, (r.kind == 'Gateway' || r.kind == 'HTTPRoute' || (r.kind == 'XListenerSet' && r.group == 'gateway.networking.x-k8s.io')) && (!has(r.group) || r.group == 'gateway.networking.k8s.io' || r.group == 'gateway.networking.x-k8s.io'))",message="targetSelectors may only reference Gateway, HTTPRoute, or XListenerSet resources"
	TargetSelectors []LocalPolicyTargetSelectorWithSectionName `json:"targetSelectors,omitempty"`

	Backend *AgentgatewayPolicyBackend `json:"backend,omitempty"`
	Frontend *AgentgatewayPolicyFrontend `json:"frontend,omitempty"`
	Traffic *AgentgatewayPolicyTraffic `json:"traffic,omitempty"`
}

type AgentgatewayPolicyBackend struct {
	TCP *BackendTCP `json:"tcp,omitempty"`
	// TLS contains the options necessary to configure a backend to use TLS origination.
	// See [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/transport_sockets/tls/v3/tls.proto#envoy-v3-api-msg-extensions-transport-sockets-tls-v3-sslconfig) for more details.
	// +optional
	TLS *TLS `json:"tls,omitempty"`
	HTTP *BackendHTTP `json:"http,omitempty"`

	MCP *BackendMCP `json:"mcp,omitempty"`
	AI *BackendAI `json:"ai,omitempty"`
	Auth *BackendAuth `json:"auth,omitempty"`

}
type AgentgatewayPolicyFrontend struct {

	HTTP *FrontendHTTP `json:"http,omitempty"`
	TLS  *FrontendTLS  `json:"tls,omitempty"`
	TCP  *FrontendTCP  `json:"tcp,omitempty"`
}
type AgentgatewayPolicyTraffic struct {

	// The phase to apply the TrafficPolicy to.
	// If the phase is Gateway, the targetRef must be a Gateway or a Listener.
	// Gateway is typically used only when a policy needs to influence the routing decision.
	//
	// Even when using Route mode, the policy can target the Gateway/Listener. This is syntax sugar for applying the policy to
	// all routes under that Gateway/Listener, and follows the merging logic described above.
	//
	// +kubebuilder:validation:Enum=Gateway;Route
	Phase string `json:"phase"`
	// Transformation is used to mutate and transform requests and responses
	// before forwarding them to the destination.
	// +optional
	Transformation *TransformationPolicy `json:"transformation,omitempty"`

	// ExtProc specifies the external processing configuration for the policy.
	// +optional
	ExtProc *ExtProcPolicy `json:"extProc,omitempty"`

	// ExtAuth specifies the external authentication configuration for the policy.
	// This controls what external server to send requests to for authentication.
	// +optional
	ExtAuth *ExtAuthPolicy `json:"extAuth,omitempty"`

	// RateLimit specifies the rate limiting configuration for the policy.
	// This controls the rate at which requests are allowed to be processed.
	// +optional
	RateLimit *RateLimit `json:"rateLimit,omitempty"`

	// Cors specifies the CORS configuration for the policy.
	// +optional
	Cors *CorsPolicy `json:"cors,omitempty"`

	// Csrf specifies the Cross-Site Request Forgery (CSRF) policy for this traffic policy.
	// +optional
	Csrf *CSRFPolicy `json:"csrf,omitempty"`

	// HeaderModifiers defines the policy to modify request and response headers.
	// +optional
	HeaderModifiers *HeaderModifiers `json:"headerModifiers,omitempty"`

	// AutoHostRewrite rewrites the Host header to the DNS name of the selected upstream.
	// NOTE: This field is only honoured for HTTPRoute targets.
	// NOTE: If `autoHostRewrite` is set on a route that also has a [URLRewrite filter](https://gateway-api.sigs.k8s.io/reference/spec/#httpurlrewritefilter)
	// configured to override the `hostname`, the `hostname` value will be used and `autoHostRewrite` will be ignored.
	// +optional
	AutoHostRewrite *bool `json:"autoHostRewrite,omitempty"`

	// Timeouts defines the timeouts for requests
	// It is applicable to HTTPRoutes and ignored for other targeted kinds.
	// +optional
	Timeouts *Timeouts `json:"timeouts,omitempty"`

	// Retry defines the policy for retrying requests.
	// It is applicable to HTTPRoutes, Gateway listeners and XListenerSets, and ignored for other targeted kinds.
	// +optional
	Retry *Retry `json:"retry,omitempty"`

	DirectResponse *DirectResponse `json:"directResponse,omitempty"`

	// RBAC specifies the role-based access control configuration for the policy.
	// This defines the rules for authorization based on roles and permissions.
	// With an Envoy-based Gateway, RBAC policies applied at different attachment points in the configuration
	// hierarchy are not cumulative, and only the most specific policy is enforced. In Envoy, this means an RBAC policy
	// attached to a route will override any RBAC policies applied to the gateway or listener. In contrast, an
	// Agentgateway-based Gateway supports cumulative RBAC policies across different attachment points, such that
	// an RBAC policy attached to a route augments policies applied to the gateway or listener without overriding them.
	Authorization *Authorization `json:"authorization,omitempty"`

	// AccessLoggingConfig contains access logging configuration
	// +kubebuilder:validation:MaxItems=16
	AccessLog []AccessLog `json:"accessLog,omitempty"`

	// Tracing contains various settings for OpenTelemetry tracer.
	// +optional
	Tracing *Tracing `json:"tracing,omitempty"`
}