package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=trafficpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=trafficpolicies/status,verbs=get;update;patch

// +kubebuilder:printcolumn:name="Accepted",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Accepted')].status",description="Traffic policy acceptance status"
// +kubebuilder:printcolumn:name="Attached",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Attached')].status",description="Traffic policy attachment status"

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels={app=kgateway,app.kubernetes.io/name=kgateway}
// +kubebuilder:resource:categories=kgateway
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="gateway.networking.k8s.io/policy=Direct"
type TrafficPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TrafficPolicySpec `json:"spec,omitempty"`

	Status gwv1.PolicyStatus `json:"status,omitempty"`
	// TODO: embed this into a typed Status field when
	// https://github.com/kubernetes/kubernetes/issues/131533 is resolved
}

// +kubebuilder:object:root=true
type TrafficPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrafficPolicy `json:"items"`
}

// TrafficPolicySpec defines the desired state of a traffic policy.
//
// TrafficPolicy can apply at two layers, based on the `phase` field:
// * After a route is selected, but before a backend is selected.
// * After a listener is selected, but before a route is selected.
//
// In either case, TrafficPolicy can target a Gateway, Listener, Route, or Route Rule.
// If multiple policies exist for a given request, they are merged on a per-field basis, with precedence to the more precise policies (Route Rule > Gateway).
//
// +kubebuilder:validation:XValidation:rule="!has(self.autoHostRewrite) || ((has(self.targetRefs) && self.targetRefs.all(r, r.kind == 'HTTPRoute')) || (has(self.targetSelectors) && self.targetSelectors.all(r, r.kind == 'HTTPRoute')))",message="autoHostRewrite can only be used when targeting HTTPRoute resources"
// +kubebuilder:validation:XValidation:rule="has(self.retry) && has(self.timeouts) ? (has(self.retry.perTryTimeout) && has(self.timeouts.request) ? duration(self.retry.perTryTimeout) < duration(self.timeouts.request) : true) : true",message="retry.perTryTimeout must be lesser than timeouts.request"
// +kubebuilder:validation:XValidation:rule="has(self.retry) && has(self.targetRefs) ? self.targetRefs.all(r, (r.kind == 'Gateway' ? has(r.sectionName) : true )) : true",message="targetRefs[].sectionName must be set when targeting Gateway resources with retry policy"
// +kubebuilder:validation:XValidation:rule="has(self.retry) && has(self.targetSelectors) ? self.targetSelectors.all(r, (r.kind == 'Gateway' ? has(r.sectionName) : true )) : true",message="targetSelectors[].sectionName must be set when targeting Gateway resources with retry policy"
type TrafficPolicySpec struct {
	// The phase to apply the TrafficPolicy to.
	// If the phase is Gateway, the targetRef must be a Gateway or a Listener.
	// Gateway is typically used only when a policy needs to influence the routing decision.
	//
	// Even when using Route mode, the policy can target the Gateway/Listener. This is syntax sugar for applying the policy to
	// all routes under that Gateway/Listener, and follows the merging logic described above.
	//
	// +kubebuilder:validation:Enum=Gateway;Route
	Phase string `json:"phase"`
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

// TransformationPolicy config is used to modify envoy behavior at a route level.
// These modifications can be performed on the request and response paths.
type TransformationPolicy struct {
	// Request is used to modify the request path.
	// +optional
	Request *Transform `json:"request,omitempty"`

	// Response is used to modify the response path.
	// +optional
	Response *Transform `json:"response,omitempty"`
}

// Transform defines the operations to be performed by the transformation.
// These operations may include changing the actual request/response but may also cause side effects.
// Side effects may include setting info that can be used in future steps (e.g. dynamic metadata) and can cause envoy to buffer.
type Transform struct {
	// Set is a list of headers and the value they should be set to.
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	Set []HeaderTransformation `json:"set,omitempty"`

	// Add is a list of headers to add to the request and what that value should be set to.
	// If there is already a header with these values then append the value as an extra entry.
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	Add []HeaderTransformation `json:"add,omitempty"`

	// Remove is a list of header names to remove from the request/response.
	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=16
	Remove []string `json:"remove,omitempty"`

	// Body controls both how to parse the body and if needed how to set.
	// If empty, body will not be buffered.
	// +optional
	Body *BodyTransformation `json:"body,omitempty"`
}

type Template string

// +kubebuilder:validation:MinLength=1
type CELExpression string

// EnvoyHeaderName is the name of a header or pseudo header
// Based on gateway api v1.Headername but allows a singular : at the start
//
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=256
// +kubebuilder:validation:Pattern=`^:?[A-Za-z0-9!#$%&'*+\-.^_\x60|~]+$`
// +k8s:deepcopy-gen=false
type (
	HeaderName           string
	HeaderTransformation struct {
		// Name is the name of the header to interact with.
		// +required
		Name HeaderName `json:"name,omitempty"`
		// Value is the template to apply to generate the output value for the header.
		// Inja templates are supported for Envoy-based data planes only.
		// CEL expressions are supported for agentgateway data plane only.
		// The system will auto-detect the appropriate template format based on the data plane.
		Value CELExpression `json:"value,omitempty"`
	}
)

// BodyparseBehavior defines how the body should be parsed
// If set to json and the body is not json then the filter will not perform the transformation.
// +kubebuilder:validation:Enum=AsString;AsJson
type BodyParseBehavior string

const (
	// BodyParseBehaviorAsString will parse the body as a string.
	BodyParseBehaviorAsString BodyParseBehavior = "AsString"
	// BodyParseBehaviorAsJSON will parse the body as a json object.
	BodyParseBehaviorAsJSON BodyParseBehavior = "AsJson"
)

// BodyTransformation controls how the body should be parsed and transformed.
type BodyTransformation struct {
	// Value is the template to apply to generate the output value for the body.
	Value CELExpression `json:"value,omitempty"`
}

// RateLimit defines a rate limiting policy.
type RateLimit struct {
	// Local defines a local rate limiting policy.
	// +optional
	Local *LocalRateLimitPolicy `json:"local,omitempty"`

	// Global defines a global rate limiting policy using an external service.
	// +optional
	Global *RateLimitPolicy `json:"global,omitempty"`
}

// LocalRateLimitPolicy represents a policy for local rate limiting.
// It defines the configuration for rate limiting using a token bucket mechanism.
type LocalRateLimitPolicy struct {
	// TokenBucket represents the configuration for a token bucket local rate-limiting mechanism.
	// It defines the parameters for controlling the rate at which requests are allowed.
	// TODO: rework this to not use TokenBucket and just use simple quotas like "100 RPS"
	// +optional
	TokenBucket *TokenBucket `json:"tokenBucket,omitempty"`
}

// TokenBucket defines the configuration for a token bucket rate-limiting mechanism.
// It controls the rate at which tokens are generated and consumed for a specific operation.
type TokenBucket struct {

	// MaxTokens specifies the maximum number of tokens that the bucket can hold.
	// This value must be greater than or equal to 1.
	// It determines the burst capacity of the rate limiter.
	// +required
	// +kubebuilder:validation:Minimum=1
	MaxTokens int32 `json:"maxTokens"`

	// TokensPerFill specifies the number of tokens added to the bucket during each fill interval.
	// If not specified, it defaults to 1.
	// This controls the steady-state rate of token generation.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	TokensPerFill *int32 `json:"tokensPerFill,omitempty"`

	// FillInterval defines the time duration between consecutive token fills.
	// This value must be a valid duration string (e.g., "1s", "500ms").
	// It determines the frequency of token replenishment.
	// +required
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	// +kubebuilder:validation:XValidation:rule="duration(self) >= duration('50ms')",message="must be at least 50ms"
	FillInterval metav1.Duration `json:"fillInterval"`
}

// RateLimitPolicy defines a global rate limiting policy using an external service.
type RateLimitPolicy struct {
	// Descriptors define the dimensions for rate limiting.
	// These values are passed to the rate limit service which applies configured limits based on them.
	// Each descriptor represents a single rate limit rule with one or more entries.
	// +required
	// +kubebuilder:validation:MinItems=1
	Descriptors []RateLimitDescriptor `json:"descriptors"`

	// ExtensionRef references a GatewayExtension that provides the global rate limit service.
	// +required
	ExtensionRef NamespacedObjectReference `json:"extensionRef"`
}

// RateLimitDescriptor defines a descriptor for rate limiting.
// A descriptor is a group of entries that form a single rate limit rule.
type RateLimitDescriptor struct {
	// Entries are the individual components that make up this descriptor.
	// When translated to Envoy, these entries combine to form a single descriptor.
	// +required
	// +kubebuilder:validation:MinItems=1
	Entries []RateLimitDescriptorEntry `json:"entries"`
}

// RateLimitDescriptorEntryType defines the type of a rate limit descriptor entry.
// +kubebuilder:validation:Enum=Generic;Header;RemoteAddress;Path
type RateLimitDescriptorEntryType string

const (
	// RateLimitDescriptorEntryTypeGeneric represents a generic key-value descriptor entry.
	RateLimitDescriptorEntryTypeGeneric RateLimitDescriptorEntryType = "Generic"

	// RateLimitDescriptorEntryTypeHeader represents a descriptor entry that extracts its value from a request header.
	RateLimitDescriptorEntryTypeHeader RateLimitDescriptorEntryType = "Header"

	// RateLimitDescriptorEntryTypeRemoteAddress represents a descriptor entry that uses the client's IP address as its value.
	RateLimitDescriptorEntryTypeRemoteAddress RateLimitDescriptorEntryType = "RemoteAddress"

	// RateLimitDescriptorEntryTypePath represents a descriptor entry that uses the request path as its value.
	RateLimitDescriptorEntryTypePath RateLimitDescriptorEntryType = "Path"
)

// RateLimitDescriptorEntry defines a single entry in a rate limit descriptor.
// Only one entry type may be specified.
// +kubebuilder:validation:XValidation:message="exactly one entry type must be specified",rule="(has(self.type) && (self.type == 'Generic' && has(self.generic) && !has(self.header)) || (self.type == 'Header' && has(self.header) && !has(self.generic)) || (self.type == 'RemoteAddress' && !has(self.generic) && !has(self.header)) || (self.type == 'Path' && !has(self.generic) && !has(self.header)))"
type RateLimitDescriptorEntry struct {
	// Key is the name of this descriptor entry.
	// +required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`

	// Value is the value for this descriptor entry.
	// +required
	Value CELExpression `json:"value"`
}

type CorsPolicy struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	*gwv1.HTTPCORSFilter `json:",inline"`

	// Disable the CORS filter.
	// Can be used to disable CORS policies applied at a higher level in the config hierarchy.
	// +optional
	Disable *PolicyDisable `json:"disable,omitempty"`
}

// CSRFPolicy can be used to set percent of requests for which the CSRF filter is enabled,
// enable shadow-only mode where policies will be evaluated and tracked, but not enforced and
// add additional source origins that will be allowed in addition to the destination origin.
//
// Dataplane Support:
// - Envoy: Supports PercentageEnabled, PercentageShadowed, and AdditionalOrigins
// - Agentgateway: Only supports AdditionalOrigins (PercentageEnabled and PercentageShadowed are ignored)
//
// +kubebuilder:validation:AtMostOneOf=percentageEnabled;percentageShadowed
type CSRFPolicy struct {
	// Specifies additional source origins that will be allowed in addition to the destination origin.
	// Envoy: Supported
	// Agentgateway: Supported
	// +optional
	// +kubebuilder:validation:MaxItems=16
	AdditionalOrigins []string `json:"additionalOrigins,omitempty"`
}

// HeaderModifiers can be used to define the policy to modify request and response headers.
// +kubebuilder:validation:XValidation:rule="has(self.request) || has(self.response)",message="At least one of request or response must be provided."
type HeaderModifiers struct {
	// Request modifies request headers.
	// +optional
	Request *gwv1.HTTPHeaderFilter `json:"request,omitempty"`

	// Response modifies response headers.
	// +optional
	Response *gwv1.HTTPHeaderFilter `json:"response,omitempty"`
}

// +kubebuilder:validation:ExactlyOneOf=maxRequestSize;disable
type Buffer struct {
	// MaxRequestSize sets the maximum size in bytes of a message body to buffer.
	// Requests exceeding this size will receive HTTP 413.
	// Example format: "1Mi", "512Ki", "1Gi"
	// +optional
	// +kubebuilder:validation:XValidation:message="maxRequestSize must be greater than 0 and less than 4Gi",rule="(type(self) == int && int(self) > 0 && int(self) < 4294967296) || (type(self) == string && quantity(self).isGreaterThan(quantity('0')) && quantity(self).isLessThan(quantity('4Gi')))"
	MaxRequestSize *resource.Quantity `json:"maxRequestSize,omitempty"`

	// Disable the buffer filter.
	// Can be used to disable buffer policies applied at a higher level in the config hierarchy.
	// +optional
	Disable *PolicyDisable `json:"disable,omitempty"`
}

// RetryOnCondition specifies the condition under which retry takes place.
//
// +kubebuilder:validation:Enum={"5xx",gateway-error,reset,reset-before-request,connect-failure,envoy-ratelimited,retriable-4xx,refused-stream,retriable-status-codes,http3-post-connect-failure,cancelled,deadline-exceeded,internal,resource-exhausted,unavailable}
type RetryOnCondition string

// Retry defines the retry policy
//
// +kubebuilder:validation:XValidation:rule="has(self.retryOn) || has(self.statusCodes)",message="retryOn or statusCodes must be set."
type Retry struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	*gwv1.HTTPCORSFilter `json:",inline"`
}

// DirectResponseSpec describes the desired state of a DirectResponse.
type DirectResponse struct {
	// StatusCode defines the HTTP status code to return for this route.
	//
	// +required
	// +kubebuilder:validation:Minimum=200
	// +kubebuilder:validation:Maximum=599
	StatusCode int32 `json:"status"`
	// Body defines the content to be returned in the HTTP response body.
	// The maximum length of the body is restricted to prevent excessively large responses.
	// If this field is omitted, no body is included in the response.
	//
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=4096
	Body *string `json:"body,omitempty"`
}
type Timeouts struct {
	// Request specifies a timeout for an individual request from the gateway to a backend.
	// This spans between the point at which the entire downstream request (i.e. end-of-stream) has been
	// processed and when the backend response has been completely processed.
	// A value of 0 effectively disables the timeout.
	// It is specified as a sequence of decimal numbers, each with optional fraction and a unit suffix, such as "1s" or "500ms".
	// +optional
	//
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	Request *metav1.Duration `json:"request,omitempty"`

	// StreamIdle specifies a timeout for a requests' idle streams.
	// A value of 0 effectively disables the timeout.
	// +optional
	//
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	StreamIdle *metav1.Duration `json:"streamIdle,omitempty"`
}
