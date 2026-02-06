package collections

import (
	apisettings "github.com/kgateway-dev/kgateway/v2/api/settings"
	"github.com/kgateway-dev/kgateway/v2/pkg/apiclient"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/krtutil"
)

type CommonCollections struct {
	Client            apiclient.Client
	KrtOpts           krtutil.KrtOptions

	// static set of global Settings, non-krt based for dev speed
	// TODO: this should be refactored to a more correct location,
	// or even better, be removed entirely and done per Gateway (maybe in GwParams)
	Settings                   apisettings.Settings
	ControllerName             string
	AgentgatewayControllerName string
}

// NewCommonCollections initializes the core krt collections.
// Collections that rely on plugins aren't initialized here,
// and InitPlugins must be called.
func NewCommonCollections(
	krtOptions krtutil.KrtOptions,
	client apiclient.Client,
	controllerName string,
	agentGatewayControllerName string,
	settings apisettings.Settings,
) (*CommonCollections, error) {
	return &CommonCollections{
		Client:            client,
		KrtOpts:           krtOptions,
		Settings:          settings,
		ControllerName:             controllerName,
		AgentgatewayControllerName: agentGatewayControllerName,
	}, nil
}
