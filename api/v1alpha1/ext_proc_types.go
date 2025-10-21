package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExtProcProvider defines the configuration for an ExtProc provider.
type ExtProcProvider struct {
	// GrpcService is the GRPC service that will handle the processing.
	// +required
	GrpcService *ExtGrpcService `json:"grpcService"`

	// FailOpen determines if requests are allowed when the ext proc service is unavailable.
	// Defaults to true, meaning requests are allowed upstream even if the ext proc service is unavailable.
	// +optional
	// +kubebuilder:default=true
	FailOpen bool `json:"failOpen,omitempty"`

	// MessageTimeout is the timeout for each message sent to the external processing server.
	// +optional
	// +kubebuilder:validation:XValidation:rule="matches(self, '^([0-9]{1,5}(h|m|s|ms)){1,4}$')",message="invalid timeout value"
	// +kubebuilder:validation:XValidation:rule="duration(self) >= duration('1ms')",message="timeout must be at least 1ms."
	MessageTimeout *metav1.Duration `json:"messageTimeout,omitempty"`

	// TODO: maybe we keep this, but needs rework for AGW.
	//MetadataOptions *MetadataOptions `json:"metadataOptions,omitempty"`
}

type ExtProcRouteCacheAction string

const (
	// RouteCacheActionFromResponse is the default behavior, which clears the route cache only
	// when the clear_route_cache field is set in an external processor response.
	RouteCacheActionFromResponse ExtProcRouteCacheAction = "FromResponse"
	// RouteCacheActionClear always clears the route cache irrespective of the
	// clear_route_cache field in the external processor response.
	RouteCacheActionClear ExtProcRouteCacheAction = "Clear"
	// RouteCacheActionRetain never clears the route cache irrespective of the
	// clear_route_cache field in the external processor response.
	RouteCacheActionRetain ExtProcRouteCacheAction = "Retain"
)

// ExtProcPolicy defines the configuration for the Envoy External Processing filter.
//
// +kubebuilder:validation:ExactlyOneOf=extensionRef;disable
type ExtProcPolicy struct {
	// ExtensionRef references the GatewayExtension that should be used for external processing.
	// +optional
	ExtensionRef *NamespacedObjectReference `json:"extensionRef,omitempty"`

	// Disable all external processing filters.
	// Can be used to disable external processing policies applied at a higher level in the config hierarchy.
	// +optional
	Disable *PolicyDisable `json:"disable,omitempty"`
}

// MetadataOptions allows configuring metadata namespaces to forward or receive from the external
// processing server.
type MetadataOptions struct {
	// Forwarding defines the typed or untyped dynamic metadata namespaces to forward to the external processing server.
	// +optional
	Forwarding *MetadataNamespaces `json:"forwarding,omitempty"`
}

// MetadataNamespaces configures which metadata namespaces to use.
// See [envoy docs](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/ext_proc/v3/ext_proc.proto#envoy-v3-api-msg-extensions-filters-http-ext-proc-v3-metadataoptions-metadatanamespaces)
// for specifics.
type MetadataNamespaces struct {
	// +optional
	// +kubebuilder:validation:MinItems=1
	Typed []string `json:"typed,omitempty"`
	// +optional
	// +kubebuilder:validation:MinItems=1
	Untyped []string `json:"untyped,omitempty"`
}

// ProcessingMode defines how the filter should interact with the request/response streams
type ProcessingMode struct {
	// RequestHeaderMode determines how to handle the request headers
	// +kubebuilder:validation:Enum=DEFAULT;SEND;SKIP
	// +kubebuilder:default=SEND
	// +optional
	RequestHeaderMode *string `json:"requestHeaderMode,omitempty"`

	// ResponseHeaderMode determines how to handle the response headers
	// +kubebuilder:validation:Enum=DEFAULT;SEND;SKIP
	// +kubebuilder:default=SEND
	// +optional
	ResponseHeaderMode *string `json:"responseHeaderMode,omitempty"`

	// RequestBodyMode determines how to handle the request body
	// +kubebuilder:validation:Enum=NONE;STREAMED;BUFFERED;BUFFERED_PARTIAL;FULL_DUPLEX_STREAMED
	// +kubebuilder:default=NONE
	// +optional
	RequestBodyMode *string `json:"requestBodyMode,omitempty"`

	// ResponseBodyMode determines how to handle the response body
	// +kubebuilder:validation:Enum=NONE;STREAMED;BUFFERED;BUFFERED_PARTIAL;FULL_DUPLEX_STREAMED
	// +kubebuilder:default=NONE
	// +optional
	ResponseBodyMode *string `json:"responseBodyMode,omitempty"`

	// RequestTrailerMode determines how to handle the request trailers
	// +kubebuilder:validation:Enum=DEFAULT;SEND;SKIP
	// +kubebuilder:default=SKIP
	// +optional
	RequestTrailerMode *string `json:"requestTrailerMode,omitempty"`

	// ResponseTrailerMode determines how to handle the response trailers
	// +kubebuilder:validation:Enum=DEFAULT;SEND;SKIP
	// +kubebuilder:default=SKIP
	// +optional
	ResponseTrailerMode *string `json:"responseTrailerMode,omitempty"`
}
