package hieraapi

import (
	"github.com/lyraproj/pcore/px"
)

type (
	DataDig   func(ctx ServerContext, key Key) px.Value
	DataHash  func(ctx ServerContext) px.OrderedMap
	LookupKey func(ctx ServerContext, key string) px.Value
)

// RegisterDataHash registers a new data_hash function with Hiera
func RegisterDataHash(name string, f DataHash) {
	px.NewGoFunction(name,
		func(d px.Dispatch) {
			d.Param(`Hiera::Context`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return f(args[0].(ServerContext))
			})
		})
}

// RegisterLookupKey registers a new lookup_key function with Hiera
func RegisterDataDig(name string, f DataDig) {
	px.NewGoFunction(name,
		func(d px.Dispatch) {
			d.Param(`Hiera::Context`)
			d.Param(`Hiera::Key`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return f(args[0].(ServerContext), args[1].(Key))
			})
		})
}

// RegisterLookupKey registers a new lookup_key function with Hiera
func RegisterLookupKey(name string, f LookupKey) {
	px.NewGoFunction(name,
		func(d px.Dispatch) {
			d.Param(`Hiera::Context`)
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return f(args[0].(ServerContext), args[1].String())
			})
		})
}
