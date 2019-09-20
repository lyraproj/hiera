package internal

import (
	"fmt"
	"sync"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/provider"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type LookupKeyProvider struct {
	hierarchyEntry hieraapi.Entry
	providerFunc   hieraapi.LookupKey
	hashes         *sync.Map
}

func (dh *LookupKeyProvider) Lookup(key hieraapi.Key, invocation hieraapi.Invocation, merge hieraapi.MergeStrategy) px.Value {
	return invocation.WithDataProvider(dh, func() px.Value {
		locations := dh.hierarchyEntry.Locations()
		switch len(locations) {
		case 0:
			return dh.invokeWithLocation(invocation, nil, key.Root())
		case 1:
			return dh.invokeWithLocation(invocation, locations[0], key.Root())
		default:
			return merge.Lookup(locations, invocation, func(location interface{}) px.Value {
				return dh.invokeWithLocation(invocation, location.(hieraapi.Location), key.Root())
			})
		}
	})
}

func (dh *LookupKeyProvider) invokeWithLocation(invocation hieraapi.Invocation, location hieraapi.Location, root string) px.Value {
	if location == nil {
		return dh.lookupKey(invocation, nil, root)
	}
	return invocation.WithLocation(location, func() px.Value {
		if location.Exists() {
			return dh.lookupKey(invocation, location, root)
		}
		invocation.ReportLocationNotFound()
		return nil
	})
}

func (dh *LookupKeyProvider) lookupKey(ic hieraapi.Invocation, location hieraapi.Location, root string) px.Value {
	key := ``
	opts := dh.hierarchyEntry.OptionsMap()
	if location != nil {
		key = location.Resolved()
		opts = optionsWithLocation(opts, key)
	}
	cache, _ := dh.hashes.LoadOrStore(key, &sync.Map{})
	value := dh.providerFunction(ic)(newServerContext(ic, cache.(*sync.Map), opts), root)
	if value != nil {
		ic.ReportFound(root, value)
	} else {
		ic.ReportNotFound(root)
	}
	return value
}

func (dh *LookupKeyProvider) providerFunction(ic hieraapi.Invocation) (pf hieraapi.LookupKey) {
	if dh.providerFunc == nil {
		dh.providerFunc = dh.loadFunction(ic)
	}
	return dh.providerFunc
}

func (dh *LookupKeyProvider) loadFunction(ic hieraapi.Invocation) (pf hieraapi.LookupKey) {
	n := dh.hierarchyEntry.Function().Name()
	switch n {
	case `environment`:
		return provider.Environment
	case `scope`:
		return provider.ScopeLookupKey
	}
	if f, ok := loadPluginFunction(ic, n, dh.hierarchyEntry); ok {
		return func(pc hieraapi.ServerContext, key string) px.Value {
			defer catchNotFound()
			return f.Call(ic, nil, []px.Value{pc.(*serverCtx), types.WrapString(key)}...)
		}
	}
	ic.ReportText(func() string { return fmt.Sprintf(`unresolved function '%s'`, n) })
	return func(pc hieraapi.ServerContext, key string) px.Value { return nil }
}

func (dh *LookupKeyProvider) FullName() string {
	return fmt.Sprintf(`lookup_key function '%s'`, dh.hierarchyEntry.Function().Name())
}

func newLookupKeyProvider(he hieraapi.Entry) hieraapi.DataProvider {
	return &LookupKeyProvider{hierarchyEntry: he, hashes: &sync.Map{}}
}
