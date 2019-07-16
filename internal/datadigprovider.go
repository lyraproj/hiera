package internal

import (
	"fmt"
	"sync"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
)

type DataDigProvider struct {
	hierarchyEntry hieraapi.Entry
	providerFunc   hieraapi.DataDig
	hashes         *sync.Map
}

func (dh *DataDigProvider) Lookup(key hieraapi.Key, invocation hieraapi.Invocation, merge hieraapi.MergeStrategy) px.Value {
	return invocation.WithDataProvider(dh, func() px.Value {
		locations := dh.hierarchyEntry.Locations()
		switch len(locations) {
		case 0:
			return dh.invokeWithLocation(invocation, nil, key)
		case 1:
			return dh.invokeWithLocation(invocation, locations[0], key)
		default:
			return merge.Lookup(locations, invocation, func(location interface{}) px.Value {
				return dh.invokeWithLocation(invocation, location.(hieraapi.Location), key)
			})
		}
	})
}

func (dh *DataDigProvider) invokeWithLocation(invocation hieraapi.Invocation, location hieraapi.Location, key hieraapi.Key) px.Value {
	if location == nil {
		return dh.lookupKey(invocation, nil, key)
	}
	result := invocation.WithLocation(location, func() px.Value {
		if location.Exists() {
			return dh.lookupKey(invocation, location, key)
		}
		invocation.ReportLocationNotFound()
		return nil
	})
	if result != nil {
		result = key.Bury(result)
	}
	return result
}

func (dh *DataDigProvider) lookupKey(ic hieraapi.Invocation, location hieraapi.Location, key hieraapi.Key) px.Value {
	cacheKey := ``
	opts := dh.hierarchyEntry.OptionsMap()
	if location != nil {
		cacheKey = location.Resolved()
		opts = optionsWithLocation(opts, cacheKey)
	}
	cache, _ := dh.hashes.LoadOrStore(cacheKey, &sync.Map{})
	value := dh.providerFunction(ic)(newProviderContext(ic, cache.(*sync.Map)), key, opts)
	if value != nil {
		ic.ReportFound(key.String(), value)
	} else {
		ic.ReportNotFound(key)
	}
	return value
}

func (dh *DataDigProvider) providerFunction(ic hieraapi.Invocation) (pf hieraapi.DataDig) {
	if dh.providerFunc == nil {
		n := dh.hierarchyEntry.Function().Name()
		// Load lookup provider function using the standard loader
		if f, ok := px.Load(ic, px.NewTypedName(px.NsFunction, n)); ok {
			dh.providerFunc = func(pc hieraapi.ProviderContext, key hieraapi.Key, options map[string]px.Value) px.Value {
				defer catchNotFound()
				return f.(px.Function).Call(ic, nil, []px.Value{pc, px.Wrap(ic, key.Parts()), px.Wrap(ic, options)}...)
			}
		} else {
			ic.ReportText(func() string {
				return fmt.Sprintf(`unresolved function '%s'`, n)
			})
			dh.providerFunc = func(pc hieraapi.ProviderContext, key hieraapi.Key, options map[string]px.Value) px.Value {
				return nil
			}
		}
	}
	return dh.providerFunc
}

func (dh *DataDigProvider) FullName() string {
	return fmt.Sprintf(`data_dig function '%s'`, dh.hierarchyEntry.Function().Name())
}

func newDataDigProvider(he hieraapi.Entry) hieraapi.DataProvider {
	return &DataDigProvider{hierarchyEntry: he, hashes: &sync.Map{}}
}
