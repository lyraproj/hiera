package lookup

import (
	"context"
	"fmt"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type LookupKind string

const KindDataDig = LookupKind(`data_dig`)
const KindDataHash = LookupKind(`data_hash`)
const KindLookupKey = LookupKind(`lookup_key`)

var FunctionKeys = []string{string(KindDataDig), string(KindDataHash), string(KindLookupKey)}

var LocationKeys = []string{string(LcPath), `paths`, string(LcGlob), `globs`, string(LcUri), `uris`, string(LcMappedPaths)}

var ReservedOptionKeys = []string{string(LcPath), string(LcUri)}

type Function interface {
	Kind() LookupKind
	Name() string
	Resolve(ic Invocation) (Function, bool)
}

type Entry interface {
	Copy(Config) Entry
	Options() px.OrderedMap
	DataDir() string
	Function() Function
}

type HierarchyEntry interface {
	Entry
	Name() string
	Resolve(ic Invocation, defaults Entry) HierarchyEntry
	CreateProvider() DataProvider
	Locations() []Location
}

type Config interface {
	Root() string
	Path() string
	LoadedConfig() px.OrderedMap
	Defaults() Entry
	Hierarchy() []HierarchyEntry
	DefaultHierarchy() []HierarchyEntry

	Resolve(ic Invocation) ResolvedConfig
}

type ResolvedConfig interface {
	// Config returns the original Config that the receiver was created from
	Config() Config

	// Hierarchy returns the DataProvider slice
	Hierarchy() []DataProvider

	// DefaultHierarchy returns the DataProvider slice for the configured default_hierarchy.
	// The slice will be empty if no such hierarchy has been defined.
	DefaultHierarchy() []DataProvider

	// LookupOptions returns the resolved lookup_options value for the given key or nil
	// if no such options exists
	LookupOptions(key Key) map[string]px.Value
}

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

// An Invocation keeps track of one specific lookup invocation implements a guard against
// endless recursion
type Invocation interface {
	px.Context

	Config() ResolvedConfig

	DoWithScope(scope px.Keyed, doer px.Doer)

	// Call doer and while it is executing, don't reveal any found values in logs
	DoRedacted(doer px.Doer)

	// NotFound should be called by a function to indicate that a specified key
	// was not found. This is different from returning an UNDEF since UNDEF is
	// a valid value for a key.
	//
	// This method will panic with an internal value that is recovered by the
	// Lookup logic. There is no return from this method.
	NotFound()

	// Execute the given function 'f' in an explanation context named by 'n'
	WithExplanationContext(n string, f func())

	// Explain will add the message returned by the given function to the
	// lookup explainer. The method will only get called when the explanation
	// support is enabled
	Explain(messageProducer func() string)

	CheckedLookup(key Key, value px.Producer) px.Value
	WithDataProvider(dh DataProvider, value px.Producer) px.Value
	WithLocation(loc Location, value px.Producer) px.Value
	ReportLocationNotFound()
	ReportFound(value px.Value)
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

var NotFound issue.Reported

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
