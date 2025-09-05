package plugins

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type AgentgatewayPlugin struct {
	AdditionalResources *AddResourcesPlugin
	ContributesPolicies map[schema.GroupKind]PolicyPlugin
	// extra has sync beyond primary resources in the collections above
	ExtraHasSynced func() bool
}

func MergePlugins(plug ...AgentgatewayPlugin) AgentgatewayPlugin {
	ret := AgentgatewayPlugin{
		ContributesPolicies: make(map[schema.GroupKind]PolicyPlugin),
	}
	var hasSynced []func() bool
	for _, p := range plug {
		// Merge contributed policies
		for gk, policy := range p.ContributesPolicies {
			ret.ContributesPolicies[gk] = policy
		}
		if p.AdditionalResources != nil {
			if ret.AdditionalResources == nil {
				ret.AdditionalResources = &AddResourcesPlugin{}
			}
			if ret.AdditionalResources.AdditionalBinds == nil {
				ret.AdditionalResources.AdditionalBinds = p.AdditionalResources.AdditionalBinds
			}
			if p.AdditionalResources.AdditionalListeners != nil {
				ret.AdditionalResources.AdditionalListeners = p.AdditionalResources.AdditionalListeners
			}
			if p.AdditionalResources.AdditionalRoutes != nil {
				ret.AdditionalResources.AdditionalRoutes = p.AdditionalResources.AdditionalRoutes
			}
			if p.AdditionalResources.AdditionalWorkloads != nil {
				ret.AdditionalResources.AdditionalWorkloads = p.AdditionalResources.AdditionalWorkloads
			}
		}
		if p.ExtraHasSynced != nil {
			hasSynced = append(hasSynced, p.ExtraHasSynced)
		}
	}
	ret.ExtraHasSynced = mergeSynced(hasSynced)
	return ret
}

func mergeSynced(funcs []func() bool) func() bool {
	return func() bool {
		for _, f := range funcs {
			if !f() {
				return false
			}
		}
		return true
	}
}

// Plugins registers all built-in policy plugins
func Plugins(agw *AgwCollections) []AgentgatewayPlugin {
	return []AgentgatewayPlugin{
		NewTrafficPlugin(agw),
		NewInferencePlugin(agw),
		NewA2APlugin(agw),
		NewBackendTLSPlugin(agw),
	}
}

func (p AgentgatewayPlugin) HasSynced() bool {
	for _, pol := range p.ContributesPolicies {
		if pol.Policies != nil && !pol.Policies.HasSynced() {
			return false
		}
	}
	if p.ExtraHasSynced != nil && !p.ExtraHasSynced() {
		return false
	}
	return true
}
