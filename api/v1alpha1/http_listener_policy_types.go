package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=httplistenerpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=httplistenerpolicies/status,verbs=get;update;patch

// +kubebuilder:printcolumn:name="Accepted",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Accepted')].status",description="HTTP listener policy acceptance status"
// +kubebuilder:printcolumn:name="Attached",type=string,JSONPath=".status.ancestors[*].conditions[?(@.type=='Attached')].status",description="HTTP listener policy attachment status"

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels={app=kgateway,app.kubernetes.io/name=kgateway}
// +kubebuilder:resource:categories=kgateway
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="gateway.networking.k8s.io/policy=Direct"
// HTTPListenerPolicy is intended to be used for configuring the Envoy `HttpConnectionManager` and any other config or policy
// that should map 1-to-1 with a given HTTP listener, such as the Envoy health check HTTP filter.
// Currently these policies can only be applied per `Gateway` but support for `Listener` attachment may be added in the future.
// See https://github.com/kgateway-dev/kgateway/issues/11786 for more details.
type FrontendPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FrontendPolicySpec `json:"spec,omitempty"`

	Status gwv1.PolicyStatus `json:"status,omitempty"`
	// TODO: embed this into a typed Status field when
	// https://github.com/kubernetes/kubernetes/issues/131533 is resolved
}

// +kubebuilder:object:root=true
type HTTPListenerPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FrontendPolicy `json:"items"`
}

// HTTPListenerPolicySpec defines the desired state of a HTTP listener policy.
type FrontendPolicySpec struct {
	// TargetRefs specifies the target resources by reference to attach the policy to.
	// This may only target a Gateway (NOT a listener!).
	// +optional
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:XValidation:rule="self.all(r, r.kind == 'Gateway' && (!has(r.group) || r.group == 'gateway.networking.k8s.io'))",message="targetRefs may only reference Gateway resources"
	TargetRefs []LocalPolicyTargetReference `json:"targetRefs,omitempty"`

	// TargetSelectors specifies the target selectors to select resources to attach the policy to.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.all(r, r.kind == 'Gateway' && (!has(r.group) || r.group == 'gateway.networking.k8s.io'))",message="targetSelectors may only reference Gateway resources"
	TargetSelectors []LocalPolicyTargetSelector `json:"targetSelectors,omitempty"`

	HTTP FrontendHTTP
	TLS FrontendTLS
	TCP FrontendTCP
}

type FrontendTCP struct {
	KeepAlive TCPKeepalive
}
type FrontendTLS struct {
	HandshakeTimeout       time.Duration  `json:"handshake_timeout"`

	// TODO: mirror the tuneables on BackendTLS
}
type FrontendHTTP struct {
	MaxBufferSize             int            `json:"max_buffer_size"`

	HTTP1MaxHeaders           *int           `json:"http1_max_headers,omitempty"`
	HTTP1IdleTimeout          time.Duration  `json:"http1_idle_timeout"`

	HTTP2WindowSize           *uint32        `json:"http2_window_size,omitempty"`
	HTTP2ConnectionWindowSize *uint32        `json:"http2_connection_window_size,omitempty"`
	HTTP2FrameSize            *uint32        `json:"http2_frame_size,omitempty"`
	HTTP2KeepaliveInterval    *time.Duration `json:"http2_keepalive_interval,omitempty"`
	HTTP2KeepaliveTimeout     *time.Duration `json:"http2_keepalive_timeout,omitempty"`
}

// AccessLog represents the top-level access log configuration.
type AccessLog struct {
	// Filter access logs configuration
	Filter CELExpression `json:"filter,omitempty"`
	Fields AccessLogFields `json:"fields,omitempty"`
}
type AccessLogFields struct {
	Remove []string
	Add map[string]CELExpression
}

// FileSink represents the file sink configuration for access logs.
// +kubebuilder:validation:ExactlyOneOf=stringFormat;jsonFormat
type FileSink struct {
	// the file path to which the file access logging service will sink
	// +required
	Path string `json:"path"`
	// the format string by which envoy will format the log lines
	// https://www.envoyproxy.io/docs/envoy/v1.33.0/configuration/observability/access_log/usage#format-strings
	StringFormat string `json:"stringFormat,omitempty"`
	// the format object by which to envoy will emit the logs in a structured way.
	// https://www.envoyproxy.io/docs/envoy/v1.33.0/configuration/observability/access_log/usage#format-dictionaries
	JsonFormat *runtime.RawExtension `json:"jsonFormat,omitempty"`
}

// AccessLogGrpcService represents the gRPC service configuration for access logs.
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/access_loggers/grpc/v3/als.proto#envoy-v3-api-msg-extensions-access-loggers-grpc-v3-httpgrpcaccesslogconfig
type AccessLogGrpcService struct {
	CommonAccessLogGrpcService `json:",inline"`

	// Additional request headers to log in the access log
	AdditionalRequestHeadersToLog []string `json:"additionalRequestHeadersToLog,omitempty"`

	// Additional response headers to log in the access log
	AdditionalResponseHeadersToLog []string `json:"additionalResponseHeadersToLog,omitempty"`

	// Additional response trailers to log in the access log
	AdditionalResponseTrailersToLog []string `json:"additionalResponseTrailersToLog,omitempty"`
}

// Common configuration for gRPC access logs.
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/access_loggers/grpc/v3/als.proto#envoy-v3-api-msg-extensions-access-loggers-grpc-v3-commongrpcaccesslogconfig
type CommonAccessLogGrpcService struct {
	CommonGrpcService `json:",inline"`

	// name of log stream
	// +required
	LogName string `json:"logName"`
}

// Common gRPC service configuration created by setting `envoy_grpc“ as the gRPC client
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/grpc_service.proto#envoy-v3-api-msg-config-core-v3-grpcservice
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/grpc_service.proto#envoy-v3-api-msg-config-core-v3-grpcservice-envoygrpc
type CommonGrpcService struct {
	// The backend gRPC service. Can be any type of supported backend (Kubernetes Service, kgateway Backend, etc..)
	// +required
	BackendRef *gwv1.BackendRef `json:"backendRef"`

	// The :authority header in the grpc request. If this field is not set, the authority header value will be cluster_name.
	// Note that this authority does not override the SNI. The SNI is provided by the transport socket of the cluster.
	// +optional
	Authority *string `json:"authority,omitempty"`

	// Maximum gRPC message size that is allowed to be received. If a message over this limit is received, the gRPC stream is terminated with the RESOURCE_EXHAUSTED error.
	// Defaults to 0, which means unlimited.
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxReceiveMessageLength *int32 `json:"maxReceiveMessageLength,omitempty"`

	// This provides gRPC client level control over envoy generated headers. If false, the header will be sent but it can be overridden by per stream option. If true, the header will be removed and can not be overridden by per stream option. Default to false.
	// +optional
	SkipEnvoyHeaders *bool `json:"skipEnvoyHeaders,omitempty"`

	// The timeout for the gRPC request. This is the timeout for a specific request
	// +optional
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Additional metadata to include in streams initiated to the GrpcService.
	// This can be used for scenarios in which additional ad hoc authorization headers (e.g. x-foo-bar: baz-key) are to be injected
	// +optional
	InitialMetadata []HeaderValue `json:"initialMetadata,omitempty"`

	// Indicates the retry policy for re-establishing the gRPC stream.
	// If max interval is not provided, it will be set to ten times the provided base interval
	// +optional
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// Header name/value pair.
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/base.proto#envoy-v3-api-msg-config-core-v3-headervalue
type HeaderValue struct {
	// Header name.
	// +required
	Key string `json:"key"`

	// Header value.
	// +optional
	Value *string `json:"value,omitempty"`
}

// Specifies the retry policy of remote data source when fetching fails.
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/base.proto#envoy-v3-api-msg-config-core-v3-retrypolicy
type RetryPolicy struct {
	// Specifies parameters that control retry backoff strategy.
	// the default base interval is 1000 milliseconds and the default maximum interval is 10 times the base interval.
	// +optional
	RetryBackOff *BackoffStrategy `json:"retryBackOff,omitempty"`

	// Specifies the allowed number of retries. Defaults to 1.
	// +optional
	// +kubebuilder:validation:Minimum=1
	NumRetries *int32 `json:"numRetries,omitempty"`
}

// Configuration defining a jittered exponential back off strategy.
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/backoff.proto#envoy-v3-api-msg-config-core-v3-backoffstrategy
type BackoffStrategy struct {
	// The base interval to be used for the next back off computation. It should be greater than zero and less than or equal to max_interval.
	// +required
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	BaseInterval metav1.Duration `json:"baseInterval"`

	// Specifies the maximum interval between retries. This parameter is optional, but must be greater than or equal to the base_interval if set. The default is 10 times the base_interval.
	// +optional
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid duration value"
	MaxInterval *metav1.Duration `json:"maxInterval,omitempty"`
}

// HeaderFilter filters requests based on headers.
// Based on: https://www.envoyproxy.io/docs/envoy/v1.33.0/api-v3/config/accesslog/v3/accesslog.proto#config-accesslog-v3-headerfilter
type HeaderFilter struct {
	// +required
	Header gwv1.HTTPHeaderMatch `json:"header"`
}

// Tracing represents the top-level Envoy's tracer.
// Ref: https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#extensions-filters-network-http-connection-manager-v3-httpconnectionmanager-tracing
type Tracing struct {
	// Provider defines the upstream to which envoy sends traces
	// +required
	Provider TracingProvider `json:"provider"`

	// Target percentage of requests managed by this HTTP connection manager that will be force traced if the x-client-trace-id header is set. Defaults to 100%
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	ClientSampling CELExpression `json:"clientSampling,omitempty"`

	// Target percentage of requests managed by this HTTP connection manager that will be randomly selected for trace generation, if not requested by the client or not forced. Defaults to 100%
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	RandomSampling CELExpression `json:"randomSampling,omitempty"`

	// A list of attributes with a unique name to create attributes for the active span.
	// +optional
	Fields []AccessLogFields `json:"fields,omitempty"`
}

// TracingProvider defines the list of providers for tracing
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type TracingProvider struct {
	// Tracing contains various settings for Envoy's OTel tracer.
	OpenTelemetry *OpenTelemetryTracingConfig `json:"openTelemetry,omitempty"`
}

// OpenTelemetryTracingConfig represents the top-level Envoy's OpenTelemetry tracer.
// See here for more information: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/trace/v3/opentelemetry.proto.html
type OpenTelemetryTracingConfig struct {
	// Send traces to the gRPC service
	// TODO: add http
	// +required
	GrpcService CommonGrpcService `json:"grpcService"`

	// The name for the service. This will be populated in the ResourceSpan Resource attributes
	// Defaults to the envoy cluster name. Ie: `<gateway-name>.<gateway-namespace>`
	// +optional
	ServiceName *string `json:"serviceName"`
}

// GrpcStatus represents possible gRPC statuses.
// +kubebuilder:validation:Enum=OK;CANCELED;UNKNOWN;INVALID_ARGUMENT;DEADLINE_EXCEEDED;NOT_FOUND;ALREADY_EXISTS;PERMISSION_DENIED;RESOURCE_EXHAUSTED;FAILED_PRECONDITION;ABORTED;OUT_OF_RANGE;UNIMPLEMENTED;INTERNAL;UNAVAILABLE;DATA_LOSS;UNAUTHENTICATED
type GrpcStatus string

// UpgradeConfig represents configuration for HTTP upgrades.
type UpgradeConfig struct {
	// List of upgrade types to enable (e.g. "websocket", "CONNECT", etc.)
	// +kubebuilder:validation:MinItems=1
	EnabledUpgrades []string `json:"enabledUpgrades,omitempty"`
}

// ServerHeaderTransformation determines how the server header is transformed.
type ServerHeaderTransformation string

const (
	// OverwriteServerHeaderTransformation overwrites the server header.
	OverwriteServerHeaderTransformation ServerHeaderTransformation = "Overwrite"
	// AppendIfAbsentServerHeaderTransformation appends to the server header if it's not present.
	AppendIfAbsentServerHeaderTransformation ServerHeaderTransformation = "AppendIfAbsent"
	// PassThroughServerHeaderTransformation passes through the server header unchanged.
	PassThroughServerHeaderTransformation ServerHeaderTransformation = "PassThrough"
)

// EnvoyHealthCheck represents configuration for Envoy's health check filter.
// The filter will be configured in No pass through mode, and will only match requests with the specified path.
type EnvoyHealthCheck struct {
	// Path defines the exact path that will be matched for health check requests.
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Pattern="^/[-a-zA-Z0-9@:%.+~#?&/=_]+$"
	Path string `json:"path"`
}
