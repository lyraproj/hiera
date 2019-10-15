package hieraapi

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

// HieraRoot is an option key that can be used to change the default root which is the current working directory
const HieraRoot = `Hiera::Root`

// HieraConfigFileName is an option that can be used to change the default file name 'hiera.yaml'
const HieraConfigFileName = `Hiera::ConfigFileName`

// HieraConfig is an option that can be used to change absolute path of the hiera config. When specified, the
// HieraRoot and HieraConfigFileName will not have any effect.
const HieraConfig = `Hiera::Config`

// HieraScope is an option that can be used to pass a variable scope to Hiera. This scope is used
// by the 'scope' lookup_key provider function and when doing variable interpolations
const HieraScope = `Hiera::Scope`

// Kind is a function kind.
type Kind string

// KindDataDig is the function kind for data_dig functions
const KindDataDig = Kind(`data_dig`)

// KindDataHash is the function kind for data_dig functions
const KindDataHash = Kind(`data_hash`)

// KindLookupKey is the function kind for data_dig functions
const KindLookupKey = Kind(`lookup_key`)

// FunctionKeys are the valid keys to use when defining a function in a hierarchy entry
var FunctionKeys = []string{string(KindDataDig), string(KindDataHash), string(KindLookupKey)}

// LocationKeys are the valid keys to use when defining locations in a hierarchy entry
var LocationKeys = []string{string(LcPath), `paths`, string(LcGlob), `globs`, string(LcURI), `uris`, string(LcMappedPaths)}

// ReservedOptionKeys are the option keys that are reserved by Hiera
var ReservedOptionKeys = []string{string(LcPath), string(LcURI)}

// A Function is a definition of a Hiera lookup function, i.e. a data_dig, data_hash, or lookup_key.
type Function interface {
	// Kind returns the function kind
	Kind() Kind

	// Name returns the name of the function
	Name() string

	// Resolve resolves the function on behalf of the given invocation
	Resolve(ic Invocation) (Function, bool)
}

// An Entry is a definition an entry in the hierarchy.
type Entry interface {
	// Create a copy of this entry for the given Config
	Copy(Config) Entry

	// Options returns the options
	Options() px.OrderedMap

	// OptionsMap returns the options as a go map
	OptionsMap() map[string]px.Value

	// DataDir returns datadir
	DataDir() string

	// PluginDir returns plugindir
	PluginDir() string

	// PluginFile returns pluginfile
	PluginFile() string

	// Function returns data_dir, data_hash, or lookup_key function
	Function() Function

	// Name returns the name
	Name() string

	// Resolve resolves this configuration on behalf of the given invocation and defaults entry
	Resolve(ic Invocation, defaults Entry) Entry

	// CreateProvider creates and returns the DataProvider configured by this entry
	CreateProvider() DataProvider

	// Locations returns the paths, globs, or uris
	Locations() []Location
}

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

	// Resolve resolves this instance into a ResolveHierarchy. Resolving means creating the proper
	// DataProviders for all Hierarchy entries
	Resolve(ic Invocation) ResolvedConfig
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
	// if no such options exists
	LookupOptions(key Key) map[string]px.Value
}

// An Invocation keeps track of one specific lookup invocation implements a guard against
// endless recursion
type Invocation interface {
	px.Context

	Config() ResolvedConfig

	DoWithScope(scope px.Keyed, doer px.Doer)

	// Call doer and while it is executing, don't reveal any found values in logs
	DoRedacted(doer px.Doer)

	// ReportText will add the message returned by the given function to the
	// lookup explainer. The method will only get called when the explanation
	// support is enabled
	ReportText(messageProducer func() string)

	// ReportLocationNotFound reports that the current location wasn't found
	ReportLocationNotFound()

	// ReportFound reports that the given value was found using the given key
	ReportFound(key interface{}, value px.Value)

	// ReportMergeResult reports the result of a the current merge operation
	ReportMergeResult(value px.Value)

	// ReportMergeSource reports the source of the current merge (explicit options or lookup options)
	ReportMergeSource(source string)

	// ReportNotFound reports that the given key was not found
	ReportNotFound(key interface{})

	// WithDataProvider pushes the given provider to the explanation stack and calls the producer, then pops the
	// provider again before returning.
	WithDataProvider(pvd DataProvider, f px.Producer) px.Value

	// WithInterpolation pushes the given expression to the explanation stack and calls the producer, then pops the
	// expression again before returning.
	WithInterpolation(expr string, f px.Producer) px.Value

	// WithInvalidKey pushes the given key to the explanation stack and calls the producer, then pops the
	// key again before returning.
	WithInvalidKey(key interface{}, f px.Producer) px.Value

	// WithLocation pushes the given location to the explanation stack and calls the producer, then pops the
	// location again before returning.
	WithLocation(loc Location, f px.Producer) px.Value

	// WithLookup pushes the given key to the explanation stack and calls the producer, then pops the
	// key again before returning.
	WithLookup(key Key, f px.Producer) px.Value

	// WithMerge pushes the given strategy to the explanation stack and calls the producer, then pops the
	// strategy again before returning.
	WithMerge(ms MergeStrategy, f px.Producer) px.Value

	// WithSegment pushes the given segment to the explanation stack and calls the producer, then pops the
	// segment again before returning.
	WithSegment(seg interface{}, f px.Producer) px.Value

	// WithLookup pushes the given key to the explanation stack and calls the producer, then pops the
	// key again before returning.
	WithSubLookup(key Key, f px.Producer) px.Value

	// ExplainMode returns true if explain support is active
	ExplainMode() bool

	// ForConfig returns an Invocation that without explainer support
	ForConfig() Invocation

	// ForData returns an Invocation that has adjusted its explainer according to
	// how it should report lookup of data as opposed to the "lookup_options" key.
	ForData() Invocation

	// ForLookupOptions returns an Invocation that has adjusted its explainer according to
	// how it should report lookup of the "lookup_options" key.
	ForLookupOptions() Invocation
}

// NotFound is the error that Hiera will panic with when a value cannot be found and no default
// value has been defined
var NotFound issue.Reported
