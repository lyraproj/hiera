package hieraapi

import (
	"github.com/lyraproj/pcore/px"
)

type (
	// DataDig performs a lookup by digging into a dotted key and returning the value that represents that key
	DataDig func(ctx ServerContext, key Key) px.Value

	// DataHash returns a hash with many values that Hiera then can use to resolve lookups
	DataHash func(ctx ServerContext) px.OrderedMap

	// LookupKey performs a lookup of the given key and returns its value.
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

// RegisterDataDig registers a new data_dig function with Hiera
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
