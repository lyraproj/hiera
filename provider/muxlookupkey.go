package provider

import (
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

const LookupProvidersKey = `hiera::lookup::providers`

// MuxLookup performs a lookup using all LookupKey function slice registered under the LookupProviderKey key
// in the given options map. The lookups are performed in the order the functions appear in the
// slice. The first found value is returned.
//
// The intended use for this function is when a very simplistic way of configuring Hiera is desired that
// requires no configuration files.
func MuxLookupKey(c hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
	if pv, ok := options[LookupProvidersKey]; ok {
		var rpv *types.RuntimeValue
		if rpv, ok = pv.(*types.RuntimeValue); ok {
			var pvs []hieraapi.LookupKey
			if pvs, ok = rpv.Interface().([]hieraapi.LookupKey); ok {
				for _, lk := range pvs {
					var result px.Value
					if result = lk(c, key, options); result != nil {
						return result
					}
				}
			}
		}
	}
	return nil
}
