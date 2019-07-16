package explain

import (
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type Context int

type Explainer interface {
	px.Value
	utils.Indentable

	// AcceptFound accepts information that a value was found for a given key
	AcceptFound(key interface{}, value px.Value)

	// AcceptFoundInDefaults accepts information that a value was found for a given key in the defaults hash
	AcceptFoundInDefaults(key string, value px.Value)

	// AcceptFoundInOverrides accepts information that a value was found for a given key in the overrides hash
	AcceptFoundInOverrides(key string, value px.Value)

	// AcceptLocationNotFound accepts information that a location was not found. The actual location is determined
	// by the top explainer node of type Context "Location"
	AcceptLocationNotFound()

	// AcceptMergeSource accepts information that about as source for merge options such as the lookup_options hash
	AcceptMergeSource(mergeSource string)

	// AcceptNotFound accepts information that a key was not found
	AcceptNotFound(key interface{})

	// AcceptResult accepts information about the result of a merge
	AcceptMergeResult(value px.Value)

	// AcceptText accepts arbitrary text to be injected into the explanation
	AcceptText(text string)

	PushDataProvider(pvd hieraapi.DataProvider)

	PushInterpolation(expr string)

	PushInvalidKey(key interface{})

	PushLocation(loc hieraapi.Location)

	PushLookup(key hieraapi.Key)

	PushMerge(mrg hieraapi.MergeStrategy)

	PushSegment(seg interface{})

	PushSubLookup(key hieraapi.Key)

	// Pop pops an explainer node from the stack of explanations
	Pop()

	// OnlyOptions returns true if lookups of lookup_options is the only thing that will be included in the explanation
	OnlyOptions() bool

	// Options returns true if lookups of lookup_options will be included in the explanation
	Options() bool
}

var NewExplainer func(options, onlyOptions bool) Explainer
