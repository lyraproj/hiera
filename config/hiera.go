package config

import (
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
)

type LookupKind string

const DataDig = LookupKind(`data_dig`)
const DataHash = LookupKind(`data_hash`)
const LookupKey = LookupKind(`lookup_key`)

var FunctionKeys = []string{string(DataDig), string(DataHash), string(LookupKey)}

var LocationKeys = []string{string(lookup.LcPath), `paths`, string(lookup.LcGlob), `globs`, string(lookup.LcUri), `uris`, string(lookup.LcMappedPaths)}

var ReservedOptionKeys = []string{string(lookup.LcPath), string(lookup.LcUri)}

type Function interface {
	Kind() LookupKind
	Name() string
	Resolve(ic lookup.Invocation) (Function, bool)
}

type Entry interface {
	Options() px.OrderedMap
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
	LoadedConfig() px.OrderedMap
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

	// ReResolve resolves the already resolved receiver using the px.Scope currently
	// held by the given px.Context. The receiver will return itself when all variables
	// in the given scope still contains the exact same values as the scope used when the
	// receiver was created,
	ReResolve(ic lookup.Invocation) ResolvedConfig
}
