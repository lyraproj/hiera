package lookup

import (
	"context"
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

// A Context provides a local cache and utility functions to a provider function
type ProviderContext interface {
	eval.PuppetObject
	eval.CallableObject

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

	// Interpolate resolves interpolations in the given value and returns the result
	Interpolate(value eval.Value) eval.Value

	// Invocation returns the active invocation.
	Invocation() Invocation
}

type Producer func() (eval.Value, bool)

// An Invocation keeps track of one specific lookup invocation implements a guard against
// endless recursion
type Invocation interface {
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

	Check(key Key, value Producer) (eval.Value, bool)
	WithDataProvider(dh DataProvider, value Producer) (eval.Value, bool)
	WithLocation(loc Location, value Producer) (eval.Value, bool)
	ReportLocationNotFound()
	ReportFound(key string, value eval.Value)
	ReportNotFound(key string)
}

// A Key is a parsed version of the possibly dot-separated key to lookup. The
// parts of a key will be strings or integers
type Key interface {
	fmt.Stringer
	Dig(eval.Value) (eval.Value, bool)
	Parts() []interface{}
	Root() string
}

type NotFound struct{}

type DataDig func(ic ProviderContext, key Key, options map[string]eval.Value) (eval.Value, bool)

type DataHash func(ic ProviderContext, options map[string]eval.Value) eval.OrderedMap

type LookupKey func(ic ProviderContext, key string, options map[string]eval.Value) (eval.Value, bool)

// TryWithParent is like eval.TryWithParent but enables lookup
var TryWithParent func(parent context.Context, tp LookupKey, options map[string]eval.Value, consumer func(eval.Context) error) error

// DoWithParent is like eval.DoWithParent but enables lookup
var DoWithParent func(parent context.Context, tp LookupKey, options map[string]eval.Value, consumer func(eval.Context))

func Lookup(ic Invocation, name string, dflt eval.Value, options map[string]eval.Value) eval.Value {
	return Lookup2(ic, []string{name}, types.DefaultAnyType(), dflt, eval.EMPTY_MAP, eval.EMPTY_MAP, options, nil)
}

var Lookup2 func(
	ic Invocation,
	names []string,
	valueType eval.Type,
	defaultValue eval.Value,
	override eval.OrderedMap,
	defaultValuesHash eval.OrderedMap,
	options map[string]eval.Value,
	block eval.Lambda) eval.Value
