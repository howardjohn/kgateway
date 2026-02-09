package deployer

import (
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	apisettings "github.com/kgateway-dev/kgateway/v2/api/settings"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/collections"
)

func NewCommonCols() *collections.CommonCollections {
	commonCols := &collections.CommonCollections{
		Settings: apisettings.Settings{
		},
	}

	return commonCols
}
