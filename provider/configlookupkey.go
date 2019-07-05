package provider

import (
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

var first = types.WrapString(`first`)

// ConfigLookupKey performs a lookup based on a hierarchy of providers that has been specified
// in a yaml based configuration stored on disk.
func ConfigLookupKey(pc hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
	ic := pc.Invocation()
	cfg := ic.Config()
	ic = ic.ForData()

	k := hieraapi.NewKey(key)
	return ic.WithLookup(k, func() px.Value {
		var lo map[string]px.Value
		merge, ok := options[`merge`]
		if ok {
			ic.ReportMergeSource(`CLI option`)
		} else {
			lo = cfg.LookupOptions(k)
			if lo == nil {
				merge = first
			} else {
				merge, ok = lo[`merge`]
				if !ok {
					merge = first
				} else {
					ic.ReportMergeSource(`"lookup_options" hash`)
				}
			}
		}

		var mh px.OrderedMap
		var mergeOpts map[string]px.Value
		if mh, ok = merge.(px.OrderedMap); ok {
			merge = mh.Get5(`strategy`, first)
			mergeOpts = make(map[string]px.Value, mh.Len())
			mh.EachPair(func(k, v px.Value) {
				ks := k.String()
				if ks != `strategy` {
					mergeOpts[ks] = v
				}
			})
		}

		redacted := false

		var convertToType px.Type
		var convertToArgs []px.Value
		if lo != nil {
			ts := ``
			if ct, ok := lo[`convert_to`]; ok {
				if cm, ok := ct.(*types.Array); ok {
					// First arg must be a type. The rest is arguments
					switch cm.Len() {
					case 0:
						// Obviously bogus
					case 1:
						ts = cm.At(0).String()
					default:
						ts = cm.At(0).String()
						convertToArgs = cm.Slice(1, cm.Len()).AppendTo(make([]px.Value, 0, cm.Len()-1))
					}
				} else {
					ts = ct.String()
				}
			}
			if ts != `` {
				convertToType = ic.ParseType(ts)
				redacted = ts == `Sensitive`
			}
		}

		var v px.Value
		hf := func() {
			ms := hieraapi.GetMergeStrategy(hieraapi.MergeStrategyName(merge.String()), mergeOpts)
			v = ms.Lookup(cfg.Hierarchy(), ic, func(prv interface{}) px.Value {
				pr := prv.(hieraapi.DataProvider)
				return pr.UncheckedLookup(k, ic, ms)
			})
		}

		if redacted {
			ic.DoRedacted(hf)
		} else {
			hf()
		}

		if v != nil && convertToType != nil {
			av := []px.Value{v}
			if convertToArgs != nil {
				av = append(av, convertToArgs...)
			}
			v = px.New(ic, convertToType, av...)
		}
		return v
	})
}
