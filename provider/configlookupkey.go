package provider

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/typ"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/merge"
	"github.com/lyraproj/hierasdk/hiera"
)

var first = vf.String(`first`)

func extractMergeInfo(sc hieraapi.ServerContext, lo dgo.Map) (mergeName dgo.Value, mergeOpts dgo.Map) {
	mergeName = sc.Option(`merge`)
	if mergeName != nil {
		sc.Invocation().ReportMergeSource(`CLI option`)
	} else {
		if lo == nil {
			mergeName = first
		} else {
			mergeName = lo.Get(`merge`)
			if mergeName == nil {
				mergeName = first
			} else {
				sc.Invocation().ReportMergeSource(`"lookup_options" hash`)
			}
		}
	}
	if mh, ok := mergeName.(dgo.Map); ok {
		if mergeName = mh.Get(`strategy`); mergeName == nil {
			mergeName = first
		}
		mergeOpts = mh.Without(`stragegy`)
	}
	return
}

func extractConversion(sc hieraapi.ServerContext, lo dgo.Map) (convertToType dgo.Type, convertToArgs dgo.Array) {
	if lo == nil {
		return
	}
	ct := lo.Get(`convert_to`)
	if ct == nil {
		return
	}
	var ts dgo.Value
	if cm, ok := ct.(dgo.Array); ok {
		// First arg must be a type. The rest is arguments
		switch cm.Len() {
		case 0:
			// Obviously bogus
		case 1:
			ts = cm.Get(0)
		default:
			ts = cm.Get(0)
			convertToArgs = cm.Slice(1, cm.Len())
		}
	} else {
		ts = ct
	}
	if ts != nil {
		ic := sc.Invocation()
		convertToType = ic.Dialect().ParseType(ic.AliasMap(), ts.(dgo.String))
	}
	return
}

// ConfigLookupKey performs a lookup based on a hierarchy of providers that has been specified
// in a yaml based configuration stored on disk.
func ConfigLookupKey(pc hiera.ProviderContext, key string) dgo.Value {
	sc, ok := pc.(hieraapi.ServerContext)
	if !ok {
		return nil
	}
	ic := sc.Invocation()
	cfg := ic.Config()
	ic = ic.ForData()

	k := hieraapi.NewKey(key)
	return ic.WithLookup(k, func() dgo.Value {
		lo := cfg.LookupOptions(k)
		mergeName, mergeOpts := extractMergeInfo(sc, lo)
		convertToType, convertToArgs := extractConversion(sc, lo)
		redacted := typ.Sensitive.Equals(convertToType)

		var v dgo.Value
		hf := func() {
			ms := merge.GetStrategy(mergeName.String(), mergeOpts)
			v = ms.MergeLookup(cfg.Hierarchy(), ic, func(prv interface{}) dgo.Value {
				pr := prv.(hieraapi.DataProvider)
				return ic.MergeLookup(k, pr, ms)
			})
		}

		if redacted {
			ic.DoRedacted(hf)
		} else {
			hf()
		}

		if v != nil && convertToType != nil {
			if convertToArgs != nil {
				v = vf.Arguments(vf.Values(v).WithAll(convertToArgs))
			}
			v = vf.New(convertToType, v)
		}
		return v
	})
}
