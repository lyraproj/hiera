package api

import (
	"github.com/lyraproj/dgo/dgo"
)

// MergeStrategy is responsible for merging or prioritizing the result of several lookups into one.
type MergeStrategy interface {
	// Label returns a short descriptive label of this strategy.
	Label() string

	// Name returns the name of this strategy
	Name() string

	// MergeLookup performs a series of lookups for each variant found in the given variants slice. The actual
	// lookup value is returned by the given value function which will be called at least once. The argument to
	// the value function will be an element of the variants slice.
	MergeLookup(variants interface{}, invocation Invocation, value func(location interface{}) dgo.Value) dgo.Value

	// Options returns the options for this strategy or an empty map if strategy has no options
	Options() dgo.Map
}
