package api

import (
	"github.com/lyraproj/dgo/dgo"
)

// An Explainer collects information about a lookup and can present it in the form of a fairly verbose
// human readable explanation.
type Explainer interface {
	dgo.Value
	dgo.Indentable

	// AcceptFound accepts information that a value was found for a given key
	AcceptFound(key interface{}, value dgo.Value)

	// AcceptFoundInDefaults accepts information that a value was found for a given key in the defaults hash
	AcceptFoundInDefaults(key string, value dgo.Value)

	// AcceptFoundInOverrides accepts information that a value was found for a given key in the overrides hash
	AcceptFoundInOverrides(key string, value dgo.Value)

	// AcceptLocationNotFound accepts information that a location was not found. The actual location is determined
	// by the top explainer node of type Context "Location"
	AcceptLocationNotFound()

	// AcceptMergeSource accepts information that about as source for merge options such as the lookup_options hash
	AcceptMergeSource(mergeSource string)

	// AcceptModuleNotFound accepts that the current module was not found
	AcceptModuleNotFound()

	// AcceptNotFound accepts information that a key was not found
	AcceptNotFound(key interface{})

	// AcceptResult accepts information about the result of a merge
	AcceptMergeResult(value dgo.Value)

	// AcceptText accepts arbitrary text to be injected into the explanation
	AcceptText(text string)

	PushDataProvider(pvd DataProvider)

	PushInterpolation(expr string)

	PushInvalidKey(key interface{})

	PushLocation(loc Location)

	PushLookup(key Key)

	PushMerge(mrg MergeStrategy)

	PushModule(moduleName string)

	PushSegment(seg interface{})

	PushSubLookup(key Key)

	// Pop pops an explainer node from the stack of explanations
	Pop()

	// OnlyOptions returns true if lookups of lookup_options is the only thing that will be included in the explanation
	OnlyOptions() bool

	// Options returns true if lookups of lookup_options will be included in the explanation
	Options() bool
}
