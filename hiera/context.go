package hiera

import (
	"context"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/hieraimpl"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
)

func NewInvocation(c px.Context, scope px.Keyed) hieraapi.Invocation {
	return hieraimpl.NewInvocation(c, scope)
}

func Lookup(ic hieraapi.Invocation, name string, dflt px.Value, options map[string]px.Value) px.Value {
	return hieraimpl.Lookup(ic, name, dflt, options)
}

// TryWithParent is like px.TryWithParent but enables lookup
func TryWithParent(parent context.Context, tp hieraapi.LookupKey, options map[string]px.Value, consumer func(px.Context) error) error {
	return pcore.TryWithParent(parent, func(c px.Context) error {
		hieraimpl.InitContext(c, tp, options)
		return consumer(c)
	})
}

// DoWithParent is like px.DoWithParent but enables lookup
func DoWithParent(parent context.Context, tp hieraapi.LookupKey, options map[string]px.Value, consumer func(px.Context)) {
	pcore.DoWithParent(parent, func(c px.Context) {
		hieraimpl.InitContext(c, tp, options)
		consumer(c)
	})
}

func Lookup2(
	ic hieraapi.Invocation,
	names []string,
	valueType px.Type,
	defaultValue px.Value,
	override px.OrderedMap,
	defaultValuesHash px.OrderedMap,
	options map[string]px.Value,
	block px.Lambda) px.Value {
	return hieraimpl.Lookup2(ic, names, valueType, defaultValue, override, defaultValuesHash, options, block)
}
