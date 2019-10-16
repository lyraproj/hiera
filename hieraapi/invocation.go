package hieraapi

import (
	"github.com/lyraproj/dgo/dgo"
)

// An Invocation keeps track of one specific lookup invocation implements a guard against
// endless recursion
type Invocation interface {
	Session

	Config() ResolvedConfig

	DoWithScope(scope dgo.Keyed, doer dgo.Doer)

	// Call doer and while it is executing, don't reveal any found values in logs
	DoRedacted(doer dgo.Doer)

	// Interpolate resolves interpolations in the given value and returns the result
	Interpolate(value dgo.Value, allowMethods bool) dgo.Value

	// InterpolateInScope resolves a key expression in the invocation scope
	InterpolateInScope(expr string, allowMethods bool) dgo.Value

	// InterpolateString resolves a string containing interpolation expressions
	InterpolateString(str string, allowMethods bool) (dgo.Value, bool)

	// Lookup performs a lookup using the given options
	Lookup(key Key, options dgo.Map) dgo.Value

	// MergeLookup performs an lookup using the given provider and merge options
	MergeLookup(key Key, provider DataProvider, merge MergeStrategy) dgo.Value

	// ReportText will add the message returned by the given function to the
	// lookup explainer. The method will only get called when the explanation
	// support is enabled
	ReportText(messageProducer func() string)

	// ReportLocationNotFound reports that the current location wasn't found
	ReportLocationNotFound()

	// ReportFound reports that the given value was found using the given key
	ReportFound(key interface{}, value dgo.Value)

	// ReportMergeResult reports the result of a the current merge operation
	ReportMergeResult(value dgo.Value)

	// ReportMergeSource reports the source of the current merge (explicit options or lookup options)
	ReportMergeSource(source string)

	// ReportNotFound reports that the given key was not found
	ReportNotFound(key interface{})

	// ServerContext returns a new server context for this invocation configured for the given entry and given options
	ServerContext(he Entry, options dgo.Map) ServerContext

	// WithDataProvider pushes the given provider to the explanation stack and calls the producer, then pops the
	// provider again before returning.
	WithDataProvider(pvd DataProvider, f dgo.Producer) dgo.Value

	// WithInterpolation pushes the given expression to the explanation stack and calls the producer, then pops the
	// expression again before returning.
	WithInterpolation(expr string, f dgo.Producer) dgo.Value

	// WithInvalidKey pushes the given key to the explanation stack and calls the producer, then pops the
	// key again before returning.
	WithInvalidKey(key interface{}, f dgo.Producer) dgo.Value

	// WithLocation pushes the given location to the explanation stack and calls the producer, then pops the
	// location again before returning.
	WithLocation(loc Location, f dgo.Producer) dgo.Value

	// WithLookup pushes the given key to the explanation stack and calls the producer, then pops the
	// key again before returning.
	WithLookup(key Key, f dgo.Producer) dgo.Value

	// WithMerge pushes the given strategy to the explanation stack and calls the producer, then pops the
	// strategy again before returning.
	WithMerge(ms MergeStrategy, f dgo.Producer) dgo.Value

	// WithSegment pushes the given segment to the explanation stack and calls the producer, then pops the
	// segment again before returning.
	WithSegment(seg interface{}, f dgo.Producer) dgo.Value

	// WithLookup pushes the given key to the explanation stack and calls the producer, then pops the
	// key again before returning.
	WithSubLookup(key Key, f dgo.Producer) dgo.Value

	// ExplainMode returns true if explain support is active
	ExplainMode() bool

	// ForConfig returns an Invocation without explain support
	ForConfig() Invocation

	// ForData returns an Invocation that has adjusted its explainer according to
	// how it should report lookup of data as opposed to the "lookup_options" key.
	ForData() Invocation

	// ForLookupOptions returns an Invocation that has adjusted its explainer according to
	// how it should report lookup of the "lookup_options" key.
	ForLookupOptions() Invocation
}
