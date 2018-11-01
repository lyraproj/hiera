package lookup

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"context"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-evaluator/types"
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
	Interpolate(val eval.PValue) eval.PValue

	// Cache adds the given key - value association to the cache
	Cache(key string, value eval.PValue) eval.PValue

	// CacheAll adds all key - value associations in the given hash to the cache
	CacheAll(hash eval.KeyedValue)

	// CachedEntry returns the value for the given key together with
	// a boolean to indicate if the value was found or not
	CachedValue(key string) (eval.PValue, bool)

	// CachedEntries calls the consumer with each entry in the cache
	CachedEntries(consumer eval.BiConsumer)
}

type lookupCtx struct {
	eval.Context
	sharedCache *ConcurrentMap
	topProvider LookupKey
	cache map[string]eval.PValue
}

// DoWithParent is like eval.DoWithParent but enables lookup
func DoWithParent(parent context.Context, provider LookupKey, consumer func(Context) error) error {
	return eval.Puppet.DoWithParent(parent, func(c eval.Context) error {
		lc := &lookupCtx{c, NewConcurrentMap(37), provider, map[string]eval.PValue{}}
		return consumer(lc)
	})
}

func Lookup(ic Invocation, name string, dflt eval.PValue, options eval.KeyedValue) eval.PValue {
	return Lookup2(ic, []string{name}, dflt, options)
}

func Lookup2(ic Invocation, names []string, dflt eval.PValue, options eval.KeyedValue) eval.PValue {
	lc := ic.Context()
	for _, name := range names {
		if v, ok := lc.(*lookupCtx).lookupViaCache(NewKey(lc, name), options); ok {
			return v
		}
	}
	if dflt == nil {
		// nil (as opposed to UNDEF) means that no default was provided.
		if len(names) == 1 {
			panic(eval.Error(lc, HIERA_NAME_NOT_FOUND, issue.H{`name`: names[0]}))
		}
		panic(eval.Error(lc, HIERA_NOT_ANY_NAME_FOUND, issue.H{`name_list`: names}))
	}
	return dflt
}

type notFound struct {}

var notFoundSingleton = &notFound{}

func (lookupCtx) NotFound() {
	panic(notFoundSingleton)
}

func (c *lookupCtx) Explain(messageProducer func() string) {
	// TODO: Add explanation support
}

func (c *lookupCtx) Interpolate(val eval.PValue) eval.PValue {
	return Interpolate(c, val, true)
}

func (c *lookupCtx) Cache(key string, value eval.PValue) eval.PValue {
	old, ok := c.cache[key]
	if !ok {
		old = eval.UNDEF
	}
	c.cache[key] = value
	return old
}

func (c *lookupCtx) CacheAll(hash eval.KeyedValue) {
	hash.EachPair(func(k, v eval.PValue) {
		c.cache[k.String()] = v
	})
}

func (c *lookupCtx) CachedValue(key string) (v eval.PValue, ok bool) {
	v, ok = c.cache[key]
	return
}

func (c *lookupCtx) CachedEntries(consumer eval.BiConsumer) {
	for k, v := range c.cache {
		consumer(types.WrapString(k), v)
	}
}

func (c *lookupCtx) WithScope(scope eval.Scope) eval.Context {
	return &lookupCtx{c.Context.WithScope(scope), c.sharedCache, c.topProvider, c.cache}
}
