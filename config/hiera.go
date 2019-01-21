package config

import (
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/puppet-evaluator/eval"
)

type LookupKind string

const DATA_DIG = LookupKind(`data_dig`)
const DATA_HASH = LookupKind(`data_hash`)
const LOOKUP_KEY = LookupKind(`lookup_key`)

var FUNCTION_KEYS = []string{string(DATA_DIG), string(DATA_HASH), string(LOOKUP_KEY)}

var LOCATION_KEYS = []string{string(lookup.LC_PATH), `paths`, string(lookup.LC_GLOB), `globs`, string(lookup.LC_URI), `uris`, string(lookup.LC_MAPPED_PATHS)}

var RESERVED_OPTION_KEYS = []string{string(lookup.LC_PATH), string(lookup.LC_URI)}

type Function interface {
	Kind() LookupKind
	Name() string
	Resolve(ic lookup.Invocation) (Function, bool)
}

type Entry interface {
	Options() eval.OrderedMap
	DataDir() string
	Function() Function
}

type HierarchyEntry interface {
	Entry
	Name() string
	Resolve(ic lookup.Invocation, defaults Entry) HierarchyEntry
	CreateProvider(ic lookup.Invocation) lookup.DataProvider
}

type Config interface {
	Root() string
	Path() string
	LoadedConfig() eval.OrderedMap
	Defaults() Entry
	Hierarchy() []HierarchyEntry
	DefaultHierarchy() []HierarchyEntry

	Resolve(ic lookup.Invocation) ResolvedConfig
}

type ResolvedConfig interface {
	// Config returns the original Config that the receiver was created from
	Config() Config

	// Hierarchy returns the DataProvider slice
	Hierarchy() []lookup.DataProvider

	// DefaultHierarchy returns the DataProvider slice for the configured default_hierarchy.
	// The slice will be empty if no such hierarchy has been defined.
	DefaultHierarchy() []lookup.DataProvider

	// ReResolve resolves the already resolved receiver using the eval.Scope currently
	// held by the given eval.Context. The receiver will return itself when all variables
	// in the given scope still contains the exact same values as the scope used when the
	// receiver was created,
	ReResolve(ic lookup.Invocation) ResolvedConfig
}
