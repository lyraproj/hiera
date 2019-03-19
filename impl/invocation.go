package impl

import (
	"github.com/lyraproj/hiera/config"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

const HieraCacheKey = `Hiera::Cache`
const HieraTopProviderKey = `Hiera::TopProvider`
const HieraGlobalOptionsKey = `Hiera::GlobalOptions`
const HieraTopProviderCacheKey = `Hiera::TopProvider::Cache`
const HieraConfigsKey = `Hiera::Config::`

type invocation struct {
	px.Context
	nameStack []string
	scope px.Keyed
}

// InitContext initializes the given context with the Hiera cache. The context initialized
// with this method determines the life-cycle of that cache.
func InitContext(c px.Context, topProvider lookup.LookupKey, options map[string]px.Value) {
	c.Set(HieraCacheKey, NewConcurrentMap(37))
	c.Set(HieraTopProviderKey, topProvider)
	c.Set(HieraTopProviderCacheKey, make(map[string]px.Value, 23))
	c.Set(HieraGlobalOptionsKey, options)
}

func NewInvocation(c px.Context, scope px.Keyed) lookup.Invocation {
	return &invocation{Context: c, nameStack: []string{}, scope: scope}
}

func (ic *invocation) topProvider() lookup.LookupKey {
	if v, ok := ic.Get(HieraTopProviderKey); ok {
		var tp lookup.LookupKey
		if tp, ok = v.(lookup.LookupKey); ok {
			return tp
		}
	}
	panic(px.Error(HIERA_NOT_INITIALIZED, issue.NoArgs))
}

func (ic *invocation) topProviderCache() map[string]px.Value {
	if v, ok := ic.Get(HieraTopProviderCacheKey); ok {
		var tc map[string]px.Value
		if tc, ok = v.(map[string]px.Value); ok {
			return tc
		}
	}
	panic(px.Error(HIERA_NOT_INITIALIZED, issue.NoArgs))
}

func (ic *invocation) globalOptions() map[string]px.Value {
	if v, ok := ic.Get(HieraGlobalOptionsKey); ok {
		var g map[string]px.Value
		if g, ok = v.(map[string]px.Value); ok {
			return g
		}
	}
	panic(px.Error(HIERA_NOT_INITIALIZED, issue.NoArgs))
}

func (ic *invocation) sharedCache() *ConcurrentMap {
	if v, ok := ic.Get(HieraCacheKey); ok {
		var sh *ConcurrentMap
		if sh, ok = v.(*ConcurrentMap); ok {
			return sh
		}
	}
	panic(px.Error(HIERA_NOT_INITIALIZED, issue.NoArgs))
}

func (ic *invocation) Config(configPath string) config.ResolvedConfig {
	val, _ := ic.sharedCache().EnsureSet(HieraConfigsKey+configPath, func() (interface{}, bool) {
		return NewConfig(ic, configPath), true
	})
	return val.(config.ResolvedConfig)
}

func (ic *invocation) lookupViaCache(key lookup.Key, options map[string]px.Value) (px.Value, bool) {
	rootKey := key.Root()

	val, ok := ic.sharedCache().EnsureSet(rootKey, func() (interface{}, bool) {
		globalOptions := ic.globalOptions()
		if len(options) == 0 {
			options = globalOptions
		} else if len(globalOptions) > 0 {
			no := make(map[string]px.Value, len(options)+len(globalOptions))
			for k, v := range globalOptions {
				no[k] = v
			}
			for k, v := range options {
				no[k] = v
			}
			options = no
		}
		if v, ok := ic.topProvider()(newContext(ic, ic.topProviderCache()), rootKey, options); ok {
			return Interpolate(ic, v, true), true
		}
		return nil, false
	})
	if ok {
		return key.Dig(val.(px.Value))
	}
	return nil, false
}

func (ic *invocation) Check(key lookup.Key, actor lookup.Producer) (px.Value, bool) {
	if utils.ContainsString(ic.nameStack, key.String()) {
		panic(px.Error(HIERA_ENDLESS_RECURSION, issue.H{`name_stack`: ic.nameStack}))
	}
	ic.nameStack = append(ic.nameStack, key.String())
	defer func() {
		ic.nameStack = ic.nameStack[:len(ic.nameStack)-1]
	}()
	return actor()
}

func (ic *invocation) DoWithScope(scope px.Keyed, doer px.Doer) {
	sc := ic.scope
	ic.scope = scope
	doer()
	ic.scope = sc
}

func (ic *invocation) Scope() px.Keyed {
	return ic.scope
}

func (ic *invocation) WithDataProvider(dh lookup.DataProvider, actor lookup.Producer) (px.Value, bool) {
	return actor()
}

func (ic *invocation) WithLocation(loc lookup.Location, actor lookup.Producer) (px.Value, bool) {
	return actor()
}

func (ic *invocation) ReportLocationNotFound() {
}

func (ic *invocation) ReportFound(key string, value px.Value) {
}

func (ic *invocation) ReportNotFound(key string) {
}

var notFoundSingleton = &lookup.NotFound{}

func (ic *invocation) NotFound() {
	panic(notFoundSingleton)
}

func (ic *invocation) Explain(messageProducer func() string) {
	// TODO: Add explanation support
}
