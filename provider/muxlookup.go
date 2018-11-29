package provider

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-hiera/lookup"
)

const LookupProvidersKey = `hiera::lookup::providers`

// MuxLookup performs a lookup using all LookupKey function slice registered under the LookupProviderKey key
// in the given options map. The lookups are performed in the order the functions appear in the
// slice. The first found value is returned.
//
// The intended use for this function is when a very simplistic way of configuring Hiera is desired that
// requires no configuration files.
func MuxLookup(c lookup.ProviderContext, key string, options eval.OrderedMap) (eval.Value, bool) {
	if pv, ok := options.Get4(LookupProvidersKey); ok {
		var rpv *types.RuntimeValue
		if rpv, ok = pv.(*types.RuntimeValue); ok {
			var pvs []lookup.LookupKey
			if pvs, ok = rpv.Interface().([]lookup.LookupKey); ok {
				for _, lk := range pvs {
					var result eval.Value
					if result, ok = lk(c, key, options); ok {
						return result, ok
					}
				}
			}
		}
	}
	return nil, false
}
