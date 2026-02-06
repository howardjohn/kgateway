package apiclient

import (
	"context"

	"istio.io/istio/pkg/config/schema/kubeclient"
	"istio.io/istio/pkg/kube/kubetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	agwv1alpha1 "github.com/kgateway-dev/kgateway/v2/api/v1alpha1/agentgateway"
	"github.com/kgateway-dev/kgateway/v2/pkg/kgateway/wellknown"
)

// RegisterTypes registers all the types used by our API Client
func RegisterTypes() {
	kubeclient.Register(
		wellknown.AgentgatewayPolicyGVR,
		wellknown.AgentgatewayPolicyGVK,
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (runtime.Object, error) {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayPolicies(namespace).List(context.Background(), o)
		},
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (watch.Interface, error) {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayPolicies(namespace).Watch(context.Background(), o)
		},
		func(c kubeclient.ClientGetter, namespace string) kubetypes.WriteAPI[*agwv1alpha1.AgentgatewayPolicy] {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayPolicies(namespace)
		},
	)
	kubeclient.Register(
		wellknown.AgentgatewayBackendGVR,
		wellknown.AgentgatewayBackendGVK,
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (runtime.Object, error) {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayBackends(namespace).List(context.Background(), o)
		},
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (watch.Interface, error) {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayBackends(namespace).Watch(context.Background(), o)
		},
		func(c kubeclient.ClientGetter, namespace string) kubetypes.WriteAPI[*agwv1alpha1.AgentgatewayBackend] {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayBackends(namespace)
		},
	)
	kubeclient.Register(
		wellknown.AgentgatewayParametersGVR,
		wellknown.AgentgatewayParametersGVK,
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (runtime.Object, error) {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayParameters(namespace).List(context.Background(), o)
		},
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (watch.Interface, error) {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayParameters(namespace).Watch(context.Background(), o)
		},
		func(c kubeclient.ClientGetter, namespace string) kubetypes.WriteAPI[*agwv1alpha1.AgentgatewayParameters] {
			return c.(Client).Kgateway().GatewayAgentgateway().AgentgatewayParameters(namespace)
		},
	)
}
