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
	function     hieraapi.Function
	locations    []hieraapi.Location
	providerFunc hieraapi.LookupKey
	hashes       *sync.Map
}

func (dh *LookupKeyProvider) UncheckedLookup(key hieraapi.Key, invocation hieraapi.Invocation, merge hieraapi.MergeStrategy) px.Value {
	return invocation.WithDataProvider(dh, func() px.Value {
		return merge.Lookup(dh.locations, invocation, func(location interface{}) px.Value {
			return dh.invokeWithLocation(invocation, location.(hieraapi.Location), key.Root())
		})
	})
}

func (dh *LookupKeyProvider) invokeWithLocation(invocation hieraapi.Invocation, location hieraapi.Location, root string) px.Value {
	if location == nil {
		return dh.lookupKey(invocation, nil, root)
	}
	return invocation.WithLocation(location, func() px.Value {
		if location.Exist() {
			return dh.lookupKey(invocation, location, root)
		}
		invocation.ReportLocationNotFound()
		return nil
	})
}

func (dh *LookupKeyProvider) lookupKey(ic hieraapi.Invocation, location hieraapi.Location, root string) px.Value {
	key := ``
	opts := NoOptions
	if location != nil {
		key = location.Resolved()
		opts = map[string]px.Value{`path`: types.WrapString(key)}
	}

	cache, _ := dh.hashes.LoadOrStore(key, &sync.Map{})
	value := dh.providerFunction(ic)(newProviderContext(ic, cache.(*sync.Map)), root, opts)
	if value != nil {
		ic.ReportFound(value)
	}
	return value
}

func (dh *LookupKeyProvider) providerFunction(ic hieraapi.Invocation) (pf hieraapi.LookupKey) {
	if dh.providerFunc == nil {
		n := dh.function.Name()
		if n == `environment` {
			dh.providerFunc = provider.Environment
		}
		// Load lookup provider function using the standard loader
		if f, ok := px.Load(ic, px.NewTypedName(px.NsFunction, n)); ok {
			dh.providerFunc = func(pc hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
				defer catchNotFound()
				return f.(px.Function).Call(ic, nil, []px.Value{pc, types.WrapString(key), px.Wrap(ic, options)}...)
			}
		} else {
			ic.Explain(func() string {
				return fmt.Sprintf(`unresolved function '%s'`, n)
			})
			dh.providerFunc = func(pc hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
				return nil
			}
		}
	}
	return dh.providerFunc
}

func (dh *LookupKeyProvider) FullName() string {
	return fmt.Sprintf(`lookup_key function '%s'`, dh.function.Name())
}

func newLookupKeyProvider(he hieraapi.HierarchyEntry) hieraapi.DataProvider {
	ls := he.Locations()
	return &LookupKeyProvider{
		function:  he.Function(),
		locations: ls,
		hashes:    &sync.Map{},
	}
}
