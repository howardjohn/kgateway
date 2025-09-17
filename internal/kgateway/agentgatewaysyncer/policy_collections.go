package agentgatewaysyncer

import (
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/plugins"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

func ADPPolicyCollection(binds krt.Collection[ADPResource], agwPlugins plugins.AgentgatewayPlugin) krt.Collection[ADPResource] {
	var allPolicies []krt.Collection[plugins.ADPPolicy]
	policyStatusMap := map[schema.GroupKind]krt.StatusCollection[controllers.Object, v1alpha2.PolicyStatus]{}
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

	allPoliciesCol := krt.NewManyCollection(joinPolicies, func(ctx krt.HandlerContext, i plugins.ADPPolicy) []ADPResource {

		res := make([]ADPResource, 0, len(uniq))
		for u := range uniq {
			logger.Debug("generating policies for gateway", "gateway", u)
			res = append(res, toResource2(u, i))
		}
		return res
	})

	return allPoliciesCol, policyStatusMap
}
