package impl

import (
	"fmt"
	"sync"

	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type DataDigProvider struct {
	function     lookup.Function
	locations    []lookup.Location
	providerFunc lookup.DataDig
	hashes       *sync.Map
}

func (dh *DataDigProvider) UncheckedLookup(key lookup.Key, invocation lookup.Invocation, merge lookup.MergeStrategy) px.Value {
	return invocation.WithDataProvider(dh, func() px.Value {
		return merge.Lookup(dh.locations, invocation, func(location interface{}) px.Value {
			return dh.invokeWithLocation(invocation, location.(lookup.Location), key)
		})
	})
}

func (dh *DataDigProvider) invokeWithLocation(invocation lookup.Invocation, location lookup.Location, key lookup.Key) px.Value {
	if location == nil {
		return dh.lookupKey(invocation, nil, key)
	}
	return invocation.WithLocation(location, func() px.Value {
		if location.Exist() {
			return dh.lookupKey(invocation, location, key)
		}
		invocation.ReportLocationNotFound()
		return nil
	})
}

func (dh *DataDigProvider) lookupKey(ic lookup.Invocation, location lookup.Location, key lookup.Key) px.Value {
	cacheKey := ``
	opts := NoOptions
	if location != nil {
		cacheKey = location.Resolved()
		opts = map[string]px.Value{`path`: types.WrapString(cacheKey)}
	}

	cache, _ := dh.hashes.LoadOrStore(cacheKey, &sync.Map{})
	if value, ok := dh.providerFunction(ic)(newProviderContext(ic, cache.(*sync.Map)), key, opts); ok {
		ic.ReportFound(value)
		return value
	}
	return nil
}

func (dh *DataDigProvider) providerFunction(ic lookup.Invocation) (pf lookup.DataDig) {
	if dh.providerFunc == nil {
		n := dh.function.Name()
		// Load lookup provider function using the standard loader
		if f, ok := px.Load(ic, px.NewTypedName(px.NsFunction, n)); ok {
			dh.providerFunc = func(pc lookup.ProviderContext, key lookup.Key, options map[string]px.Value) (px.Value, bool) {
				v := f.(px.Function).Call(ic, nil, []px.Value{pc, px.Wrap(ic, key.Parts()), px.Wrap(ic, options)}...)
				return v, v != nil
			}
		} else {
			ic.Explain(func() string {
				return fmt.Sprintf(`unresolved function '%s'`, n)
			})
			dh.providerFunc = func(pc lookup.ProviderContext, key lookup.Key, options map[string]px.Value) (px.Value, bool) {
				return nil, false
			}
		}
	}
	return dh.providerFunc
}

func (dh *DataDigProvider) FullName() string {
	return fmt.Sprintf(`data_dig function '%s'`, dh.function.Name())
}

func newDataDigProvider(he lookup.HierarchyEntry) lookup.DataProvider {
	ls := he.Locations()
	return &DataDigProvider{
		function:  he.Function(),
		locations: ls,
		hashes:    &sync.Map{},
	}
}
