package agentgatewaysyncer

import (
	"log"
	"strings"

	"github.com/agentgateway/agentgateway/go/api"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/slices"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/plugins"
	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/translator"
	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/utils"
	"github.com/kgateway-dev/kgateway/v2/pkg/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/krtutil"
)

type PolicyStatusCollections = map[schema.GroupKind]krt.StatusCollection[controllers.Object, gwv1.PolicyStatus]

func AgwPolicyCollection(agwPlugins plugins.AgwPlugin, ancestors krt.IndexCollection[utils.TypedNamespacedName, *utils.AncestorBackend], krtopts krtutil.KrtOptions) (krt.Collection[ir.AgwResource], PolicyStatusCollections) {
	var allPolicies []krt.Collection[plugins.AgwPolicy]
	policyStatusMap := PolicyStatusCollections{}
	// Collect all policies from registered plugins.
	// Note: Only one plugin should be used per source GVK.
	// Avoid joining collections per-GVK before passing them to a plugin.
	for gvk, plugin := range agwPlugins.ContributesPolicies {
		policy, policyStatus := plugin.ApplyPolicies(plugins.PolicyPluginInput{Ancestors: ancestors})
		allPolicies = append(allPolicies, policy)
		if policyStatus != nil {
			// some plugins may not have a status collection (a2a services, etc.)
			policyStatusMap[gvk] = policyStatus
		}
	}
	joinPolicies := krt.JoinCollection(allPolicies, krtopts.ToOptions("JoinPolicies")...)

	allPoliciesCol := krt.NewManyCollection(joinPolicies, func(ctx krt.HandlerContext, i plugins.AgwPolicy) []ir.AgwResource {
		tgt := i.Policy.Target
		switch tt := tgt.Kind.(type) {
		case *api.PolicyTarget_Gateway:
			return []ir.AgwResource{translator.ToResourceForGateway(types.NamespacedName{
				Namespace: tt.Gateway.Namespace,
				Name:      tt.Gateway.Name,
			}, i)}
		case *api.PolicyTarget_Route:
			// TODO: implement a Route <--> Gateway lookup. Note we need to encode the `kind` of the route into the proto, which we need to do for other reasons.
			return []ir.AgwResource{translator.ToResourceGlobal(i)}
		case *api.PolicyTarget_Backend:
			key := utils.TypedNamespacedName{
				NamespacedName: types.NamespacedName{
					Namespace: tt.Backend.Namespace,
					Name:      tt.Backend.Name,
				},
				Kind: wellknown.AgentgatewayBackendGVK.Kind,
			}
			gateways := krt.FetchOne(ctx, ancestors, krt.FilterKey(key.String()))
			return slices.Map(gateways.Objects, func(gw *utils.AncestorBackend) ir.AgwResource {
				return translator.ToResourceForGateway(gw.Gateway, i)
			})
		case *api.PolicyTarget_Service:
			name, _, _ := strings.Cut(tt.Service.Hostname, ".")
			key := utils.TypedNamespacedName{
				NamespacedName: types.NamespacedName{
					Namespace: tt.Service.Namespace,
					Name:      name,
				},
				Kind: wellknown.ServiceGVK.Kind,
			}
			gateways := krt.FetchOne(ctx, ancestors, krt.FilterKey(key.String()))
			return slices.Map(gateways.Objects, func(gw *utils.AncestorBackend) ir.AgwResource {
				return translator.ToResourceForGateway(gw.Gateway, i)
			})
		default:
			log.Fatalf("unknown policy target type: %T", tt)
			return nil
		}
	}, krtopts.ToOptions("AllPolicies")...)

	return allPoliciesCol, policyStatusMap
}
