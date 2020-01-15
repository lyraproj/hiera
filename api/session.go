package api

import (
	"context"
	"sync"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/streamer"
	"github.com/lyraproj/hierasdk/hiera"
)

// A Session determines the life cycle of cached values during a Hiera session.
type Session interface {
	context.Context

	// AliasMap is the map that manages all type aliases used during the session.
	AliasMap() dgo.AliasMap

	// Dialect determines what language to use when parsing types and serializing/deserializing
	// rich data.
	Dialect() streamer.Dialect

	// KillPlugins ensures that all external plugin processes that were started by this session are killed.
	KillPlugins()

	// LoadFunction loads the lookup function defined in the given hierarchy entry and returns
	// it together with a flag indicating if the load was a success
	LoadFunction(he Entry) (dgo.Function, bool)

	// Invocation creates a new invocation for this session
	Invocation(scope interface{}, explainer Explainer) Invocation

	// SessionOptions returns the session specific options
	SessionOptions() dgo.Map

	// Loader returns the session specific loader
	Loader() dgo.Loader

	// Scope returns the session's scope
	Scope() dgo.Keyed

	// SharedCache returns the cache that is shared
	SharedCache() *sync.Map

	// TopProvider returns the lookup function that defines the hierarchy
	TopProvider() hiera.LookupKey

	// TopProviderCache returns the shared provider cache used by all lookups
	TopProviderCache() *sync.Map

	// Get returns a session variable, or nil if no such variable exists. Session variables
	// are used internally by Hiera and should not be confused with Scope variables.
	Get(key string) interface{}
}
