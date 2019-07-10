package hieraapi

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type MergeStrategyName string

const (
	First  = MergeStrategyName(`first`)
	Unique = MergeStrategyName(`unique`)
	Hash   = MergeStrategyName(`hash`)
	Deep   = MergeStrategyName(`deep`)
)

// GetMergeStrategy returns the MergeStrategy that corresponds to the given name. The
// options argument is only applicable to deep merge
var GetMergeStrategy func(name MergeStrategyName, options map[string]px.Value) MergeStrategy

// MergeStrategy is responsible for merging or prioritizing the result of several lookups into one.
type MergeStrategy interface {
	issue.Labeled

	// Lookup performs a series of lookups for each variant found in the given variants slice. The actual
	// lookup value is returned by the given value function which will be called at least once. The argument to
	// the value function will be an element of the variants slice.
	Lookup(variants interface{}, invocation Invocation, value func(location interface{}) px.Value) px.Value

	// Options returns the options for this strategy or an empty map if strategy has no options
	Options() px.OrderedMap
}
