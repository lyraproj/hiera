package hieraapi

import (
	"github.com/lyraproj/pcore/px"
)

// GetMergeStrategy returns the MergeStrategy that corresponds to the given name. The
// options argument is only applicable to deep merge
var GetMergeStrategy func(name string, options map[string]px.Value) MergeStrategy

// MergeStrategy is responsible for merging or prioritizing the result of several lookups into one.
type MergeStrategy interface {
	// Lookup performs a series of lookups for each variant found in the given variants slice. The actual
	// lookup value is returned by the given value function which will be called at least once. The argument to
	// the value function will be an element of the variants slice.
	Lookup(variants interface{}, invocation Invocation, value func(location interface{}) px.Value) px.Value
}
