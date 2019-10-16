package provider

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
)

const LookupKeyFunctions = `hiera::lookup::providers`

// MuxLookup performs a lookup using all LookupKey function slice registered under the LookupProviderKey key
// in the given options map. The lookups are performed in the order the functions appear in the
// slice. The first found value is returned.
//
// The intended use for this function is when a very simplistic way of configuring Hiera is desired that
// requires no configuration files.
func MuxLookupKey(pc hiera.ProviderContext, key string) dgo.Value {
	iv := pc.(hieraapi.ServerContext).Invocation()
	if pv := iv.SessionOptions().Get(LookupKeyFunctions); pv != nil {
		if rpv, ok := pv.(dgo.Array); ok {
			args := vf.MutableValues(pc, key)
			found := rpv.Find(func(e dgo.Value) interface{} {
				if lk, ok := e.(dgo.Function); ok {
					return lk.Call(args)[0]
				}
				return nil
			})
			if found != nil {
				return vf.Value(found)
			}
		}
	}
	return nil
}
