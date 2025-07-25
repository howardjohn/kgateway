package cache

import (
	"context"
	"errors"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/xds"
)

type CacheKey struct {
	Type string
	Node string
}

type EnvoySnapshot struct {
	Snapshot      envoycache.SnapshotCache
	mu            sync.Mutex
	perNodeTypes  sets.Set[string]
	perNodeLinear map[CacheKey]*envoycache.LinearCache
	perTypeTypes  sets.Set[string]
	perTypeLinear map[string]*envoycache.LinearCache
	Hasher        envoycache.NodeHash
}

func (mux *EnvoySnapshot) classify(url string, node *core.Node) CacheKey {
	return CacheKey{
		Type: url,
		Node: mux.Hasher.ID(node),
	}
}

func (mux *EnvoySnapshot) CreateWatch(request *envoycache.Request, state stream.StreamState, value chan envoycache.Response) func() {
	if lc := mux.forKey(mux.classify(request.TypeUrl, request.Node)); lc != nil {
		return lc.CreateWatch(request, state, value)
	}
	return mux.Snapshot.CreateWatch(request, state, value)
}

func (mux *EnvoySnapshot) CreateDeltaWatch(request *envoycache.DeltaRequest, state stream.StreamState, value chan envoycache.DeltaResponse) func() {
	if lc := mux.forKey(mux.classify(request.TypeUrl, request.Node)); lc != nil {
		return lc.CreateDeltaWatch(request, state, value)
	}
	return mux.Snapshot.CreateDeltaWatch(request, state, value)
}

func (mux *EnvoySnapshot) Fetch(context.Context, *envoycache.Request) (envoycache.Response, error) {
	return nil, errors.New("not implemented")
}

func (mux *EnvoySnapshot) RegisterPerType(url string) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.perTypeTypes.Insert(url)
}

func (mux *EnvoySnapshot) RegisterPerNode(url string) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.perNodeTypes.Insert(url)
}

func (mux *EnvoySnapshot) For(url string, node string) *envoycache.LinearCache {
	k := CacheKey{
		Type: url,
		Node: node,
	}
	return mux.forKey(k)
}

func (mux *EnvoySnapshot) forKey(k CacheKey) *envoycache.LinearCache {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.perNodeTypes.Has(k.Type) {
		if v, f := mux.perNodeLinear[k]; f {
			return v
		}
		lc := envoycache.NewLinearCache(k.Type)
		mux.perNodeLinear[k] = lc
		return lc
	}
	if mux.perTypeTypes.Has(k.Type) {
		if v, f := mux.perTypeLinear[k.Type]; f {
			return v
		}
		lc := envoycache.NewLinearCache(k.Type)
		mux.perTypeLinear[k.Type] = lc
		return lc
	}
	return nil
}

var _ envoycache.Cache = &EnvoySnapshot{}

func New(log log.Logger) *EnvoySnapshot {
	snapshotCache := envoycache.NewSnapshotCache(true, xds.NewNodeRoleHasher(), log)
	return &EnvoySnapshot{
		Hasher:        xds.NewNodeRoleHasher(),
		Snapshot:      snapshotCache,
		perTypeTypes:  make(sets.Set[string]),
		perTypeLinear: map[string]*envoycache.LinearCache{},
		perNodeTypes:  make(sets.Set[string]),
		perNodeLinear: map[CacheKey]*envoycache.LinearCache{},
	}
}
