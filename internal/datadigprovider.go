package internal

import (
	"fmt"
	"sync"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type DataDigProvider struct {
	function     hieraapi.Function
	locations    []hieraapi.Location
	providerFunc hieraapi.DataDig
	hashes       *sync.Map
}

func (dh *DataDigProvider) UncheckedLookup(key hieraapi.Key, invocation hieraapi.Invocation, merge hieraapi.MergeStrategy) px.Value {
	return invocation.WithDataProvider(dh, func() px.Value {
		return merge.Lookup(dh.locations, invocation, func(location interface{}) px.Value {
			return dh.invokeWithLocation(invocation, location.(hieraapi.Location), key)
		})
	})
}

func (dh *DataDigProvider) invokeWithLocation(invocation hieraapi.Invocation, location hieraapi.Location, key hieraapi.Key) px.Value {
	var v px.Value
	if location == nil {
		v = dh.lookupKey(invocation, nil, key)
	} else {
		v = invocation.WithLocation(location, func() px.Value {
			if location.Exist() {
				return dh.lookupKey(invocation, location, key)
			}
			invocation.ReportLocationNotFound()
			return nil
		})
	}
	if v != nil {
		v = key.Bury(v)
	}
	return v
}

func (dh *DataDigProvider) lookupKey(ic hieraapi.Invocation, location hieraapi.Location, key hieraapi.Key) px.Value {
	cacheKey := ``
	opts := NoOptions
	if location != nil {
		cacheKey = location.Resolved()
		opts = map[string]px.Value{`path`: types.WrapString(cacheKey)}
	}

	cache, _ := dh.hashes.LoadOrStore(cacheKey, &sync.Map{})
	value := dh.providerFunction(ic)(newProviderContext(ic, cache.(*sync.Map)), key, opts)
	if value != nil {
		ic.ReportFound(value)
	}
	return value
}

func (dh *DataDigProvider) providerFunction(ic hieraapi.Invocation) (pf hieraapi.DataDig) {
	if dh.providerFunc == nil {
		n := dh.function.Name()
		// Load lookup provider function using the standard loader
		if f, ok := px.Load(ic, px.NewTypedName(px.NsFunction, n)); ok {
			dh.providerFunc = func(pc hieraapi.ProviderContext, key hieraapi.Key, options map[string]px.Value) px.Value {
				defer catchNotFound()
				return f.(px.Function).Call(ic, nil, []px.Value{pc, px.Wrap(ic, key.Parts()), px.Wrap(ic, options)}...)
			}
		} else {
			ic.Explain(func() string {
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
	return fmt.Sprintf(`data_dig function '%s'`, dh.function.Name())
}

func newDataDigProvider(he hieraapi.HierarchyEntry) hieraapi.DataProvider {
	ls := he.Locations()
	return &DataDigProvider{
		function:  he.Function(),
		locations: ls,
		hashes:    &sync.Map{},
	}
}
