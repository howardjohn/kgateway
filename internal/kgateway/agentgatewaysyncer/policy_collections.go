package agentgatewaysyncer

import (
	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/ir"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/ptr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/plugins"
)

type PolicyStatusCollections = map[schema.GroupKind]krt.StatusCollection[controllers.Object, v1alpha2.PolicyStatus]

func ADPPolicyCollection(agwPlugins plugins.AgentgatewayPlugin) (krt.Collection[ir.ADPResource], PolicyStatusCollections) {
	var allPolicies []krt.Collection[plugins.ADPPolicy]
	policyStatusMap := PolicyStatusCollections{}
	// Collect all policies from registered plugins.
	// Note: Only one plugin should be used per source GVK.
	// Avoid joining collections per-GVK before passing them to a plugin.
	for gvk, plugin := range agwPlugins.ContributesPolicies {
		policy, policyStatus := plugin.ApplyPolicies()
		allPolicies = append(allPolicies, policy)
		if policyStatus != nil {
			// some plugins may not have a status collection (a2a services, etc.)
			policyStatusMap[gvk] = policyStatus
		}
	}
	joinPolicies := krt.JoinCollection(allPolicies, krt.WithName("AllPolicies"))

	allPoliciesCol := krt.NewCollection(joinPolicies, func(ctx krt.HandlerContext, i plugins.ADPPolicy) *ir.ADPResource {
		return ptr.Of(toResourceGlobal(i))
	})

	return allPoliciesCol, policyStatusMap
}
