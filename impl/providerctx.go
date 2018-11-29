package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-hiera/lookup"
	"io"
)

var ContextType eval.ObjectType

func init() {
	ContextType = eval.NewObjectType(`Puppet::LookupContext`, `{
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
	cache       map[string]eval.Value
}

func (c *providerCtx) Interpolate(value eval.Value) eval.Value {
	return Interpolate(c.invocation, value, true)
}

func newContext(c lookup.Invocation, cache map[string]eval.Value) lookup.ProviderContext {
	// TODO: Cache should be specific to a provider identity determined by the providers position in
	//  the configured hierarchy
	return &providerCtx{invocation: c, cache: cache}
}

func (c *providerCtx) Call(ctx eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (result eval.Value, ok bool) {
	result = eval.UNDEF
	ok = true
	switch method.Name() {
	case `cache`:
		result = c.Cache(args[0].String(), args[1])
	case `cache_all`:
		c.CacheAll(args[0].(eval.OrderedMap))
	case `cached_value`:
		if v, ok := c.CachedValue(args[0].String()); ok {
			result = v
		}
	case `cached_entries`:
		c.CachedEntries(func(k, v eval.Value) { block.Call(ctx, nil, k, v)})
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
	return eval.ToString(c)
}

func (c *providerCtx) Equals(other interface{}, guard eval.Guard) bool {
	return c == other
}

func (c *providerCtx) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(c, s, b, g)
}

func (c *providerCtx) PType() eval.Type {
	return ContextType
}

func (c *providerCtx) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `environment_name`, `module_name`:
		return eval.UNDEF, true
	}
	return nil, false
}

func (c *providerCtx) InitHash() eval.OrderedMap {
	return eval.EMPTY_MAP
}

func (c *providerCtx) NotFound() {
	c.invocation.NotFound()
}

func (c *providerCtx) Explain(messageProducer func() string) {
	c.invocation.Explain(messageProducer)
}

func (c *providerCtx) Cache(key string, value eval.Value) eval.Value {
	old, ok := c.cache[key]
	if !ok {
		old = eval.UNDEF
	}
	c.cache[key] = value
	return old
}

func (c *providerCtx) CacheAll(hash eval.OrderedMap) {
	hash.EachPair(func(k, v eval.Value) {
		c.cache[k.String()] = v
	})
}

func (c *providerCtx) CachedValue(key string) (v eval.Value, ok bool) {
	v, ok = c.cache[key]
	return
}

func (c *providerCtx) CachedEntries(consumer eval.BiConsumer) {
	for k, v := range c.cache {
		consumer(types.WrapString(k), v)
	}
}
