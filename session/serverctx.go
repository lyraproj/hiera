package session

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hierasdk/hiera"
)

type serverCtx struct {
	hiera.ProviderContext
	invocation api.Invocation
}

func (c *serverCtx) Interpolate(value dgo.Value) dgo.Value {
	return c.invocation.Interpolate(value, true)
}

func (c *serverCtx) Explain(messageProducer func() string) {
	c.invocation.ReportText(messageProducer)
}

func (c *serverCtx) Cache(key string, value dgo.Value) dgo.Value {
	cache := c.invocation.TopProviderCache()
	if old, loaded := cache.LoadOrStore(key, value); loaded {
		// Replace old value
		cache.Store(key, value)
		return old.(dgo.Value)
	}
	return nil
}

func (c *serverCtx) CacheAll(hash dgo.Map) {
	cache := c.invocation.TopProviderCache()
	hash.EachEntry(func(e dgo.MapEntry) {
		cache.Store(e.Key().String(), e.Value())
	})
}

func (c *serverCtx) CachedValue(key string) (dgo.Value, bool) {
	if v, ok := c.invocation.TopProviderCache().Load(key); ok {
		return v.(dgo.Value), true
	}
	return nil, false
}

func (c *serverCtx) CachedEntries(consumer func(key string, value dgo.Value)) {
	c.invocation.TopProviderCache().Range(func(k, v interface{}) bool {
		consumer(k.(string), v.(dgo.Value))
		return true
	})
}

func (c *serverCtx) Invocation() api.Invocation {
	return c.invocation
}

func (c *serverCtx) ForData() api.ServerContext {
	return &serverCtx{ProviderContext: c.ProviderContext, invocation: c.invocation.ForData()}
}

func (c *serverCtx) ForLookupOptions() api.ServerContext {
	return &serverCtx{ProviderContext: c.ProviderContext, invocation: c.invocation.ForLookupOptions()}
}
