package plugins

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agentgateway/agentgateway/go/api"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/ptr"
	"istio.io/istio/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/utils"
	"github.com/kgateway-dev/kgateway/v2/pkg/kgateway/translator/sslutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/kubeutils"
)

// NewBackendTLSPlugin creates a new BackendTLSPolicy plugin
func NewBackendTLSPlugin(agw *AgwCollections) AgwPlugin {
	return AgwPlugin{
		ContributesPolicies: map[schema.GroupKind]PolicyPlugin{
			wellknown.BackendTLSPolicyGVK.GroupKind(): {
				Build: func(input PolicyPluginInput) (krt.StatusCollection[controllers.Object, gwv1.PolicyStatus], krt.Collection[AgwPolicy]) {
					st, o := krt.NewStatusManyCollection(agw.BackendTLSPolicies, func(krtctx krt.HandlerContext, btls *gwv1.BackendTLSPolicy) (*gwv1.PolicyStatus, []AgwPolicy) {
						return translatePoliciesForBackendTLS(krtctx, agw.ControllerName, input.Ancestors, agw.ConfigMaps, btls)
					}, agw.KrtOpts.ToOptions("agentgateway/BackendTLSPolicy")...)
					return convertStatusCollection(st), o
				},
			},
		},
	}
}

// translatePoliciesForService generates backend TLS policies
func translatePoliciesForBackendTLS(
	krtctx krt.HandlerContext,
	controllerName string,
	ancestors krt.IndexCollection[utils.TypedNamespacedName, *utils.AncestorBackend],
	cfgmaps krt.Collection[*corev1.ConfigMap],
	btls *gwv1.BackendTLSPolicy,
) (*gwv1.PolicyStatus, []AgwPolicy) {
	logger := logger.With("plugin_kind", "backendtls")
	var policies []AgwPolicy
	status := btls.Status.DeepCopy()

	// Condition reporting for BackendTLSPolicy is tricky. The references are to Service (or other backends), but we report
	// per-gateway.
	// This means most of the results are aggregated.
	conds := map[string]*condition{
		string(gwv1.PolicyConditionAccepted): {
			reason:  string(gwv1.PolicyReasonAccepted),
			message: "Configuration is valid",
		},
		string(gwv1.BackendTLSPolicyConditionResolvedRefs): {
			reason:  string(gwv1.BackendTLSPolicyReasonResolvedRefs),
			message: "Configuration is valid",
		},
	}

	caCert, err := getBackendTLSCACert(krtctx, cfgmaps, btls)
	if err != nil {
		logger.Error("error getting backend TLS CA cert", "policy", kubeutils.NamespacedNameFrom(btls), "error", err)
		conds[string(gwv1.PolicyConditionAccepted)].error = &ConfigError{
			Reason:  string(gwv1.PolicyReasonInvalid),
			Message: err.Error(),
		}
		// a sentinel value to send to agentgateway to signal that it should reject TLS connects due to invalid config
		caCert = []byte("invalid")
	}

	// Ideally we would report status for an unknown reference. However, Gateway API has decided we should report 1 status
	// per Gateway, instead of per-Backend. This is questionable for users, but also means we don't have to worry about
	// telling users if a reference is invalid and should just silently fail...
	uniqueGateways := sets.New[types.NamespacedName]()
	for _, target := range btls.Spec.TargetRefs {
		var policyTarget *api.PolicyTarget

		tgtRef := utils.TypedNamespacedName{
			NamespacedName: types.NamespacedName{
				Name:      string(target.Name),
				Namespace: btls.Namespace,
			},
			Kind: string(target.Kind),
		}
		ancestorBackends := krt.Fetch(krtctx, ancestors, krt.FilterKey(tgtRef.String()))
		for _, gwl := range ancestorBackends {
			for _, i := range gwl.Objects {
				uniqueGateways.Insert(i.Gateway)
			}
		}
		switch string(target.Kind) {
		case wellknown.AgentgatewayBackendGVK.Kind:
			policyTarget = &api.PolicyTarget{
				Kind: utils.BackendTarget(btls.Namespace, string(target.Name), target.SectionName),
			}
		case wellknown.ServiceKind:
			policyTarget = &api.PolicyTarget{
				Kind: utils.ServiceTarget(btls.Namespace, string(target.Name), (*string)(target.SectionName)),
			}
		case wellknown.InferencePoolKind:
			policyTarget = &api.PolicyTarget{
				Kind: utils.InferencePoolTarget(btls.Namespace, string(target.Name), (*string)(target.SectionName)),
			}
		default:
			logger.Warn("unsupported target kind", "kind", target.Kind, "policy", btls.Name)
			continue
		}
		policy := &api.Policy{
			Key:    btls.Namespace + "/" + btls.Name + backendTlsPolicySuffix + attachmentName(policyTarget),
			Name:   TypedResourceName(wellknown.BackendTLSPolicyKind, btls),
			Target: policyTarget,
			Kind: &api.Policy_Backend{
				Backend: &api.BackendPolicySpec{
					Kind: &api.BackendPolicySpec_BackendTls{
						BackendTls: &api.BackendPolicySpec_BackendTLS{
							Root: caCert,
							// Used for mTLS, not part of the spec currently
							Cert: nil,
							Key:  nil,
							// Validation.Hostname is a required value and validated with CEL
							Hostname: ptr.Of(string(btls.Spec.Validation.Hostname)),
						},
					},
				},
			},
		}
		policies = append(policies, AgwPolicy{policy})
	}
	ancestorStatus := make([]gwv1.PolicyAncestorStatus, 0, len(btls.Spec.TargetRefs))
	for g := range uniqueGateways {
		pr := gwv1.ParentReference{
			Group: ptr.Of(gwv1.Group(gvk.KubernetesGateway.Group)),
			Kind:  ptr.Of(gwv1.Kind(gvk.KubernetesGateway.Kind)),
			Name:  gwv1.ObjectName(g.Name),
		}
		ancestorStatus = append(ancestorStatus, setAncestorStatus(pr, status, btls.Generation, conds, gwv1.GatewayController(controllerName)))
	}
	status.Ancestors = mergeAncestors(controllerName, status.Ancestors, ancestorStatus)
	return status, policies
}

func getBackendTLSCACert(
	krtctx krt.HandlerContext,
	cfgmaps krt.Collection[*corev1.ConfigMap],
	btls *gwv1.BackendTLSPolicy,
) ([]byte, error) {
	validation := btls.Spec.Validation
	if wk := validation.WellKnownCACertificates; wk != nil {
		switch kind := *wk; kind {
		case gwv1.WellKnownCACertificatesSystem:
			return nil, nil

		default:
			return nil, fmt.Errorf("unsupported wellKnownCACertificates: %v", kind)
		}
	}

	// One of WellKnownCACertificates or CACertificateRefs will always be specified (CEL validated)
	if len(validation.CACertificateRefs) == 0 {
		// should never happen as this is CEL validated. Only here to prevent panic in tests
		return nil, errors.New("BackendTLSPolicy must specify either wellKnownCACertificates or caCertificateRefs")
	}
	var sb strings.Builder
	for _, ref := range validation.CACertificateRefs {
		if ref.Group != gwv1.Group(wellknown.ConfigMapGVK.Group) || ref.Kind != gwv1.Kind(wellknown.ConfigMapGVK.Kind) {
			return nil, fmt.Errorf("BackendTLSPolicy's validation.caCertificateRefs must be a ConfigMap reference; got %s", ref)
		}
		nn := types.NamespacedName{
			Name:      string(ref.Name),
			Namespace: btls.Namespace,
		}
		cfgmap := krt.FetchOne(krtctx, cfgmaps, krt.FilterObjectName(nn))
		if cfgmap == nil {
			return nil, fmt.Errorf("ConfigMap %s not found", nn)
		}
		caCert, err := sslutils.GetCACertFromConfigMap(ptr.Flatten(cfgmap))
		if err != nil {
			return nil, fmt.Errorf("error extracting CA cert from ConfigMap %s: %w", nn, err)
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(caCert)
	}
	return []byte(sb.String()), nil
}
