package lookup

import (
	"context"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
)

// A Context is passed to a configured lookup data provider function. The
// context is guaranteed to be unique for the given function in the configuration
// where its declared.
type Context interface {
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
	// Parent context
	eval.Context
	sharedCache *ConcurrentMap
	topProvider LookupKey
	cache map[string]eval.Value
}

// TryWithParent is like eval.TryWithParent but enables lookup
func TryWithParent(parent context.Context, provider LookupKey, consumer func(Context) error) error {
	return eval.Puppet.TryWithParent(parent, func(c eval.Context) error {
		lc := &lookupCtx{c, NewConcurrentMap(37), provider, map[string]eval.Value{}}
		if _, ok := parent.(*lookupCtx); !ok {
			InitContext(lc)
		}
		return consumer(lc)
	})
}
// DoWithParent is like eval.DoWithParent but enables lookup
func DoWithParent(parent context.Context, provider LookupKey, consumer func(Context)) {
	eval.Puppet.DoWithParent(parent, func(c eval.Context) {
		lc := &lookupCtx{c, NewConcurrentMap(37), provider, map[string]eval.Value{}}
		if _, ok := parent.(*lookupCtx); !ok {
			InitContext(lc)
		}
		consumer(lc)
	})
}


func Lookup(ic Invocation, name string, dflt eval.Value, options eval.OrderedMap) eval.Value {
	return Lookup2(ic, []string{name}, types.DefaultAnyType(), dflt, eval.EMPTY_MAP, eval.EMPTY_MAP, options, nil)
}

func Lookup2(
  ic Invocation,
	names []string,
	valueType eval.Type,
	defaultValue eval.Value,
	override eval.OrderedMap,
	defaultValuesHash eval.OrderedMap,
	options eval.OrderedMap,
	block eval.Lambda) eval.Value {
	for _, name := range names {
		if v, ok := ic.(*invocation).lookupViaCache(NewKey(name), options); ok {
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
