package api

import (
	"github.com/lyraproj/dgo/dgo"
)

// A Config represents a full hiera.yaml version 5 configuration.
type Config interface {
	// Root returns the directory holding this Config
	Root() string

	// Path is the full path to this Config
	Path() string

	// Defaults returns the Defaults entry
	Defaults() Entry

	// Hierarchy returns the configuration hierarchy slice
	Hierarchy() []Entry

	// DefaultHierarchy returns the default hierarchy slice
	DefaultHierarchy() []Entry
}

// A ResolvedConfig represents a Config where everything has been resolved on behalf of an Invocation.
type ResolvedConfig interface {
	// Config returns the original Config that the receiver was created from
	Config() Config

	// Hierarchy returns the DataProvider slice
	Hierarchy() []DataProvider

	// DefaultHierarchy returns the DataProvider slice for the configured default_hierarchy.
	// The slice will be empty if no such hierarchy has been defined.
	DefaultHierarchy() []DataProvider

	// LookupOptions returns the resolved lookup_options value for the given key or nil
	// if no such options exists.
	LookupOptions(key Key) dgo.Map
}
