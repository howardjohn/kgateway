package plugins

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/agentgateway/agentgateway/go/api"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/kubeutils"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/ptr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/logging"
)

// NewBackendTLSPlugin creates a new A2A policy plugin
func NewBackendTLSPlugin(agw *AgwCollections) AgentgatewayPlugin {
	domainSuffix := kubeutils.GetClusterDomainName()
	policyCol := krt.NewManyCollection(agw.BackendTLSPolicies, func(krtctx krt.HandlerContext, btls *gwv1alpha3.BackendTLSPolicy) []ADPPolicy {
		return translatePoliciesForBackendTLS(krtctx, agw.ConfigMaps, btls, domainSuffix)
	})
	return AgentgatewayPlugin{
		ContributesPolicies: map[schema.GroupKind]PolicyPlugin{
			wellknown.BackendTLSPolicyGVK.GroupKind(): {
				Policies: policyCol,
			},
		},
		ExtraHasSynced: func() bool {
			return policyCol.HasSynced()
		},
	}
}

// translatePoliciesForService generates A2A policies for a single service
func translatePoliciesForBackendTLS(krtctx krt.HandlerContext,
	cfgmaps krt.Collection[*corev1.ConfigMap],
	btls *gwv1alpha3.BackendTLSPolicy, domainSuffix string) []ADPPolicy {
	logger := logging.New("agentgateway/plugins/backendtls")
	var policies []ADPPolicy

	for idx, target := range btls.Spec.TargetRefs {
		var policyTarget *api.PolicyTarget

		switch string(target.Kind) {
		case wellknown.BackendGVK.Kind:
			policyTarget = &api.PolicyTarget{
				Kind: &api.PolicyTarget_Backend{
					Backend: btls.Namespace + "/" + string(target.Name),
				},
			}
		case wellknown.ServiceKind:
			// 'service/{namespace}/{hostname}:{port}'
			svc := fmt.Sprintf("service/%v/%v.%v.svc.%v",
				btls.Namespace, target.Name, btls.Namespace, domainSuffix)
			// TODO: allow port-less attachment in agentgateway
			if s := target.SectionName; s != nil {
				// TODO: validate it is a port?
				svc += ":" + string(*s)
			}
			policyTarget = &api.PolicyTarget{
				Kind: &api.PolicyTarget_Backend{
					Backend: svc,
				},
			}

		default:
			logger.Warn("unsupported target kind", "kind", target.Kind, "policy", btls.Name)
			continue
		}

		// TODO: support btls.Spec.Validation.Hostname.
		// Needs AGW support.

		policy := &api.Policy{
			Name:   btls.Namespace + "/" + btls.Name + ":" + strconv.Itoa(idx) + ":backendtls",
			Target: policyTarget,
			Spec: &api.PolicySpec{Kind: &api.PolicySpec_BackendTls{
				BackendTls: &api.PolicySpec_BackendTLS{
					Root: wrapperspb.Bytes([]byte(getBackendTLSCredentialName(krtctx, cfgmaps, btls))),
					// Used for mTLS, not part of the spec currently
					Cert: nil,
					Key:  nil,
					// Not currently in the spec.
					Insecure: nil,
				},
			}},
		}
		policies = append(policies, ADPPolicy{policy})
	}

	return policies
}

func getBackendTLSCredentialName(
	krtctx krt.HandlerContext,
	cfgmaps krt.Collection[*corev1.ConfigMap],
	btls *gwv1alpha3.BackendTLSPolicy,
) string {
	validation := btls.Spec.Validation
	if wk := validation.WellKnownCACertificates; wk != nil {
		switch *wk {
		case gwv1alpha3.WellKnownCACertificatesSystem:
			// Already our default, no action needed
		default:
			// TODO: report status
		}
		return ""
	}
	if len(validation.CACertificateRefs) == 0 {
		return ""
	}

	// Spec should require but double check
	// We only support 1
	cacerts := []string{}
	for _, ref := range validation.CACertificateRefs {
		nn := types.NamespacedName{
			Name:      string(ref.Name),
			Namespace: btls.Namespace,
		}
		// TODO: make sure its a configmap reference, reject others
		cfgmap := krt.FetchOne(krtctx, cfgmaps, krt.FilterObjectName(nn))
		if cfgmap == nil {
			// TODO: error
			continue
		}
		cacert, err := extractCARoot(ptr.Flatten(cfgmap))
		if err != nil {
			// TODO: error
			continue
		}
		cacerts = append(cacerts, cacert)
	}
	return ""
}

func extractCARoot(cm *corev1.ConfigMap) (string, error) {
	caCrt, ok := cm.Data["ca.crt"]
	if !ok {
		return "", errors.New("ca.crt key missing")
	}

	return caCrt, nil
}
