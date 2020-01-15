package api

import (
	"github.com/lyraproj/dgo/dgo"
)

// A DataProvider performs a lookup using a configured lookup function.
type DataProvider interface {
	// Hierarchy returns the entry where this provider was configured
	Hierarchy() Entry

	// FullName returns a descriptive name of the data provider. Used by the explainer
	FullName() string

	// Perform a lookup of the given key, invocation and location, and return the result.
	// The invocation is guaranteed to be a resolved locations derived from the locations
	// present in this providers hierarchy, or nil if no location is present.
	LookupKey(key Key, ic Invocation, location Location) dgo.Value
}
