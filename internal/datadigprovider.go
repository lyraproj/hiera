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
	value := dh.providerFunction(ic)(newServerContext(ic, cache.(*sync.Map), opts), key)
	if value != nil {
		ic.ReportFound(key.Source(), value)
	} else {
		ic.ReportNotFound(key)
	}
	return value
}

func (dh *DataDigProvider) providerFunction(ic hieraapi.Invocation) (pf hieraapi.DataDig) {
	if dh.providerFunc == nil {
		dh.providerFunc = dh.loadFunction(ic)
	}
	return dh.providerFunc
}

func (dh *DataDigProvider) loadFunction(ic hieraapi.Invocation) (pf hieraapi.DataDig) {
	n := dh.hierarchyEntry.Function().Name()
	if f, ok := loadPluginFunction(ic, n, dh.hierarchyEntry); ok {
		return func(pc hieraapi.ServerContext, key hieraapi.Key) px.Value {
			defer catchNotFound()
			return f.(px.Function).Call(ic, nil, []px.Value{pc.(*serverCtx), key}...)
		}
	}
	ic.ReportText(func() string { return fmt.Sprintf(`unresolved function '%s'`, n) })
	return func(pc hieraapi.ServerContext, key hieraapi.Key) px.Value { return nil }
}

func (dh *DataDigProvider) FullName() string {
	return fmt.Sprintf(`data_dig function '%s'`, dh.hierarchyEntry.Function().Name())
}

func newDataDigProvider(he hieraapi.Entry) hieraapi.DataProvider {
	return &DataDigProvider{hierarchyEntry: he, hashes: &sync.Map{}}
}
