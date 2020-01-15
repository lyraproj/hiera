package api

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/hierasdk/hiera"
)

// ServerContext is the Hiera context used by lookup functions that operate in-process
type ServerContext interface {
	hiera.ProviderContext

	// ReportText will add the message returned by the given function to the
	// lookup explainer. The method will only get called when the explanation
	// support is enabled
	Explain(messageProducer func() string)

	// Cache adds the given key - value association to the cache
	Cache(key string, value dgo.Value) dgo.Value

	// CacheAll adds all key - value associations in the given hash to the cache
	CacheAll(hash dgo.Map)

	// CachedEntry returns the value for the given key together with
	// a boolean to indicate if the value was found or not
	CachedValue(key string) (dgo.Value, bool)

	// CachedEntries calls the consumer with each association in the cache
	CachedEntries(consumer func(key string, value dgo.Value))

	// Interpolate resolves interpolations in the given value and returns the result
	Interpolate(value dgo.Value) dgo.Value

	// Invocation returns the active invocation.
	Invocation() Invocation

	// Returns a copy of this ServerContext with an Invocation that is configured for lookup of data
	ForData() ServerContext

	// Returns a copy of this ServerContext with an Invocation that is configured for lookup of lookup_options
	ForLookupOptions() ServerContext
}
