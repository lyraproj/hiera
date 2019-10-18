package hieraapi

import "github.com/lyraproj/pcore/px"

// DeepMerge will merge the values 'a' and 'b' if both values are hashes or both values are
// arrays. When this is not the case, no merge takes place and the 'a' argument is returned.
// The second bool return value true if a merge took place and false when the first argument
// is returned.
//
// When both values are hashes, DeepMerge is called recursively entries with identical keys.
// When both values are arrays, the merge creates a union of the unique elements from the two arrays.
// No recursive merge takes place for the array elements.
var DeepMerge func(a, b px.Value, options map[string]px.Value) (px.Value, bool)
