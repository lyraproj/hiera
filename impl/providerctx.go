package impl

import (
	"io"

	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

var ContextType px.ObjectType

func init() {
	ContextType = px.NewObjectType(`Hiera::Context`, `{
    attributes => {
      environment_name => {
        type => String[1],
        kind => derived
      },
      module_name => {
        type => Optional[String[1]],
        kind => derived
      }
    },
    functions => {
      not_found => Callable[[0,0], Undef],
      explain => Callable[[Callable[0, 0]], Undef],
      interpolate => Callable[1, 1],
      cache => Callable[[Scalar, Any], Any],
      cache_all => Callable[[Hash[Scalar, Any]], Undef],
      cache_has_key => Callable[[Scalar], Boolean],
      cached_value => Callable[[Scalar], Any],
      cached_entries => Variant[
        Callable[[Callable[1,1]], Undef],
        Callable[[Callable[2,2]], Undef],
        Callable[[0, 0], Iterable[Tuple[Scalar, Any]]]],
      cached_file_data => Callable[String,Optional[Callable[1,1]]],
    }
  }`)
}

type providerCtx struct {
	invocation lookup.Invocation
	cache      map[string]px.Value
}

func (c *providerCtx) Interpolate(value px.Value) px.Value {
	return Interpolate(c.invocation, value, true)
}

func newContext(c *invocation, cache map[string]px.Value) lookup.ProviderContext {
	// TODO: Cache should be specific to a provider identity determined by the providers position in
	//  the configured hierarchy
	return &providerCtx{invocation: c, cache: cache}
}

func (c *providerCtx) Call(ctx px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	result = px.Undef
	ok = true
	switch method.Name() {
	case `cache`:
		result = c.Cache(args[0].String(), args[1])
	case `cache_all`:
		c.CacheAll(args[0].(px.OrderedMap))
	case `cached_value`:
		if v, ok := c.CachedValue(args[0].String()); ok {
			result = v
		}
	case `cached_entries`:
		c.CachedEntries(func(k, v px.Value) { block.Call(ctx, nil, k, v) })
	case `explain`:
		c.Explain(func() string { return block.Call(ctx, nil).String() })
	case `not_found`:
		c.NotFound()
	default:
		result = nil
		ok = false
	}
	return result, ok
}

func (c *providerCtx) String() string {
	return px.ToString(c)
}

func (c *providerCtx) Equals(other interface{}, guard px.Guard) bool {
	return c == other
}

func (c *providerCtx) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	types.ObjectToString(c, s, b, g)
}

func (c *providerCtx) PType() px.Type {
	return ContextType
}

func (c *providerCtx) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `environment_name`, `module_name`:
		return px.Undef, true
	}
	return nil, false
}

func (c *providerCtx) InitHash() px.OrderedMap {
	return px.EmptyMap
}

func (c *providerCtx) NotFound() {
	c.invocation.NotFound()
}

func (c *providerCtx) Explain(messageProducer func() string) {
	c.invocation.Explain(messageProducer)
}

func (c *providerCtx) Cache(key string, value px.Value) px.Value {
	old, ok := c.cache[key]
	if !ok {
		old = px.Undef
	}
	c.cache[key] = value
	return old
}

func (c *providerCtx) CacheAll(hash px.OrderedMap) {
	hash.EachPair(func(k, v px.Value) {
		c.cache[k.String()] = v
	})
}

func (c *providerCtx) CachedValue(key string) (v px.Value, ok bool) {
	v, ok = c.cache[key]
	return
}

func (c *providerCtx) CachedEntries(consumer px.BiConsumer) {
	for k, v := range c.cache {
		consumer(types.WrapString(k), v)
	}
}

func (c *providerCtx) Invocation() lookup.Invocation {
	return c.invocation
}
