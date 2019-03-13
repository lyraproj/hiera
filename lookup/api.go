package lookup

import (
	"context"
	"fmt"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// A Context provides a local cache and utility functions to a provider function
type ProviderContext interface {
	px.PuppetObject
	px.CallableObject

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
	Cache(key string, value px.Value) px.Value

	// CacheAll adds all key - value associations in the given hash to the cache
	CacheAll(hash px.OrderedMap)

	// CachedEntry returns the value for the given key together with
	// a boolean to indicate if the value was found or not
	CachedValue(key string) (px.Value, bool)

	// CachedEntries calls the consumer with each entry in the cache
	CachedEntries(consumer px.BiConsumer)

	// Interpolate resolves interpolations in the given value and returns the result
	Interpolate(value px.Value) px.Value

	// Invocation returns the active invocation.
	Invocation() Invocation
}

type Producer func() (px.Value, bool)

// An Invocation keeps track of one specific lookup invocation implements a guard against
// endless recursion
type Invocation interface {
	px.Context

	DoWithScope(scope px.Keyed, doer px.Doer)

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

	Check(key Key, value Producer) (px.Value, bool)
	WithDataProvider(dh DataProvider, value Producer) (px.Value, bool)
	WithLocation(loc Location, value Producer) (px.Value, bool)
	ReportLocationNotFound()
	ReportFound(key string, value px.Value)
	ReportNotFound(key string)
}

// A Key is a parsed version of the possibly dot-separated key to lookup. The
// parts of a key will be strings or integers
type Key interface {
	fmt.Stringer
	Dig(px.Value) (px.Value, bool)
	Parts() []interface{}
	Root() string
}

type NotFound struct{}

type DataDig func(ic ProviderContext, key Key, options map[string]px.Value) (px.Value, bool)

type DataHash func(ic ProviderContext, options map[string]px.Value) px.OrderedMap

type LookupKey func(ic ProviderContext, key string, options map[string]px.Value) (px.Value, bool)

// TryWithParent is like px.TryWithParent but enables lookup
var TryWithParent func(parent context.Context, tp LookupKey, options map[string]px.Value, consumer func(px.Context) error) error

// DoWithParent is like px.DoWithParent but enables lookup
var DoWithParent func(parent context.Context, tp LookupKey, options map[string]px.Value, consumer func(px.Context))

func Lookup(ic Invocation, name string, dflt px.Value, options map[string]px.Value) px.Value {
	return Lookup2(ic, []string{name}, types.DefaultAnyType(), dflt, px.EmptyMap, px.EmptyMap, options, nil)
}

var Lookup2 func(
	ic Invocation,
	names []string,
	valueType px.Type,
	defaultValue px.Value,
	override px.OrderedMap,
	defaultValuesHash px.OrderedMap,
	options map[string]px.Value,
	block px.Lambda) px.Value
