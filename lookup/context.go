package lookup

import (
	"context"
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
)

// A Context is passed to a configured lookup data provider function. The
// context is guaranteed to be unique for the given function in the configuration
// where its declared.
type Context interface {
	// Parent context
	eval.Context

	// NotFound should be called by a function to indicate that a specified key
	// was not found. This is different from returning an UNDEF since UNDEF is
	// a valid value for a key.
	//
	// This method will panic with an internal value that is recovered by the
	// Lookup logic. There is no return from this method.
	NotFound()

	// Explain will add the message returned by the given function to the
	// lookup explainer. The method will only get called when the explanation
	// support is enabled
	Explain(messageProducer func() string)

	// Interpolate resolves interpolation expressions in the given value and returns
	// the result
	Interpolate(val eval.Value) eval.Value

	// Cache adds the given key - value association to the cache
	Cache(key string, value eval.Value) eval.Value

	// CacheAll adds all key - value associations in the given hash to the cache
	CacheAll(hash eval.OrderedMap)

	// CachedEntry returns the value for the given key together with
	// a boolean to indicate if the value was found or not
	CachedValue(key string) (eval.Value, bool)

	// CachedEntries calls the consumer with each entry in the cache
	CachedEntries(consumer eval.BiConsumer)
}

type lookupCtx struct {
	eval.Context
	sharedCache *ConcurrentMap
	topProvider LookupKey
	cache map[string]eval.Value
}

// DoWithParent is like eval.DoWithParent but enables lookup
func DoWithParent(parent context.Context, provider LookupKey, consumer func(Context) error) error {
	return eval.Puppet.DoWithParent(parent, func(c eval.Context) error {
		lc := &lookupCtx{c, NewConcurrentMap(37), provider, map[string]eval.Value{}}
		return consumer(lc)
	})
}

func Lookup(c eval.Context, name string, dflt eval.Value, options eval.OrderedMap) eval.Value {
	return Lookup2(c, []string{name}, types.DefaultAnyType(), dflt, eval.EMPTY_MAP, eval.EMPTY_MAP, options, nil)
}

func Lookup2(
	ctx eval.Context,
	names []string,
	valueType eval.Type,
	defaultValue eval.Value,
	override eval.OrderedMap,
	defaultValuesHash eval.OrderedMap,
	options eval.OrderedMap,
	block eval.Lambda) eval.Value {
	lc, ok := ctx.(*lookupCtx)
	if !ok {
		panic(fmt.Errorf(`lookup called without lookup.Context`))
	}
	for _, name := range names {
		if v, ok := lc.lookupViaCache(NewKey(name), options); ok {
			return v
		}
	}
	if defaultValue == nil {
		// nil (as opposed to UNDEF) means that no default was provided.
		if len(names) == 1 {
			panic(eval.Error(HIERA_NAME_NOT_FOUND, issue.H{`name`: names[0]}))
		}
		panic(eval.Error(HIERA_NOT_ANY_NAME_FOUND, issue.H{`name_list`: names}))
	}
	return defaultValue
}

type notFound struct {}

var notFoundSingleton = &notFound{}

func (lookupCtx) NotFound() {
	panic(notFoundSingleton)
}

func (c *lookupCtx) Explain(messageProducer func() string) {
	// TODO: Add explanation support
}

func (c *lookupCtx) Interpolate(val eval.Value) eval.Value {
	return Interpolate(c, val, true)
}

func (c *lookupCtx) Cache(key string, value eval.Value) eval.Value {
	old, ok := c.cache[key]
	if !ok {
		old = eval.UNDEF
	}
	c.cache[key] = value
	return old
}

func (c *lookupCtx) CacheAll(hash eval.OrderedMap) {
	hash.EachPair(func(k, v eval.Value) {
		c.cache[k.String()] = v
	})
}

func (c *lookupCtx) CachedValue(key string) (v eval.Value, ok bool) {
	v, ok = c.cache[key]
	return
}

func (c *lookupCtx) CachedEntries(consumer eval.BiConsumer) {
	for k, v := range c.cache {
		consumer(types.WrapString(k), v)
	}
}

func (c *lookupCtx) Fork() eval.Context {
	return &lookupCtx{
		Context: c.Context.Fork(),
		sharedCache: c.sharedCache,
		topProvider: c.topProvider,
		cache: map[string]eval.Value{},
	}
}

func (c *lookupCtx) WithScope(scope eval.Scope) eval.Context {
	return &lookupCtx{c.Context.WithScope(scope), c.sharedCache, c.topProvider, c.cache}
}

func (c *lookupCtx) lookupViaCache(key Key, options eval.OrderedMap) (eval.Value, bool) {
	rootKey := key.Root()

	val := c.sharedCache.EnsureSet(rootKey, func() (val interface{}) {
		defer func() {
			if r := recover(); r != nil {
				if r == notFoundSingleton {
					val = r
				} else {
					panic(r)
				}
			}
		}()
		val = Interpolate(c, c.topProvider(c, rootKey, options), true)
		return
	})
	if val == notFoundSingleton {
		return nil, false
	}
	return key.Dig(val.(eval.Value))
}