package provider

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/merge"
	"github.com/lyraproj/hierasdk/hiera"
)

// LookupKeyFunctions is the key that the MuxLookupKey function will use when finding the functions
// that it delegates to.
const LookupKeyFunctions = `hiera::lookup::providers`

// MuxLookupKey performs a lookup using all LookupKey function slice registered under the LookupProviderKey key
// in the given options map. The lookups are performed in the order the functions appear in the
// slice. The first found value is returned.
//
// The intended use for this function is when a very simplistic way of configuring Hiera is desired that
// requires no configuration files.
func MuxLookupKey(pc hiera.ProviderContext, key string) dgo.Value {
	sc, ok := pc.(api.ServerContext)
	ic := sc.Invocation()
	if !ok {
		return nil
	}
	var rpv dgo.Array
	if rpv, ok = ic.SessionOptions().Get(LookupKeyFunctions).(dgo.Array); !ok {
		return nil
	}
	spv := rpv.AppendToSlice(make([]dgo.Value, 0, rpv.Len()))

	luSc := sc.ForLookupOptions()
	args := vf.MutableValues(luSc, key)
	luFunc := func(pv interface{}) dgo.Value {
		if lk, ok := pv.(dgo.Function); ok {
			return lk.Call(args)[0]
		}
		return nil
	}
	luOpts, _ := merge.GetStrategy(`deep`, nil).MergeLookup(spv, luSc.Invocation(), luFunc).(dgo.Map)

	sc = sc.ForData()
	args.Set(0, sc)
	ic = sc.Invocation()
	return ic.WithLookup(api.NewKey(key), func() dgo.Value {
		ic.SetMergeStrategy(sc.Option(`merge`), luOpts)
		return ic.LookupAndConvertData(func() dgo.Value {
			return ic.MergeStrategy().MergeLookup(spv, ic, luFunc)
		})
	})
}
