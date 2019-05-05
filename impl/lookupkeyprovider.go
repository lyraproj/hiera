package impl

import (
	"fmt"
	"sync"

	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type LookupKeyProvider struct {
	function     lookup.Function
	locations    []lookup.Location
	providerFunc lookup.LookupKey
	hashes       *sync.Map
}

func (dh *LookupKeyProvider) UncheckedLookup(key lookup.Key, invocation lookup.Invocation, merge lookup.MergeStrategy) px.Value {
	return invocation.WithDataProvider(dh, func() px.Value {
		return merge.Lookup(dh.locations, invocation, func(location interface{}) px.Value {
			return dh.invokeWithLocation(invocation, location.(lookup.Location), key.Root())
		})
	})
}

func (dh *LookupKeyProvider) invokeWithLocation(invocation lookup.Invocation, location lookup.Location, root string) px.Value {
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

func (dh *LookupKeyProvider) lookupKey(ic lookup.Invocation, location lookup.Location, root string) px.Value {
	key := ``
	opts := NoOptions
	if location != nil {
		key = location.Resolved()
		opts = map[string]px.Value{`path`: types.WrapString(key)}
	}

	cache, _ := dh.hashes.LoadOrStore(key, &sync.Map{})
	if value, ok := dh.providerFunction(ic)(newProviderContext(ic, cache.(*sync.Map)), root, opts); ok {
		ic.ReportFound(value)
		return value
	}
	return nil
}

func (dh *LookupKeyProvider) providerFunction(ic lookup.Invocation) (pf lookup.LookupKey) {
	if dh.providerFunc == nil {
		n := dh.function.Name()
		// Load lookup provider function using the standard loader
		if f, ok := px.Load(ic, px.NewTypedName(px.NsFunction, n)); ok {
			dh.providerFunc = func(pc lookup.ProviderContext, key string, options map[string]px.Value) (px.Value, bool) {
				v := f.(px.Function).Call(ic, nil, []px.Value{pc, types.WrapString(key), px.Wrap(ic, options)}...)
				return v, v != nil
			}
		} else {
			ic.Explain(func() string {
				return fmt.Sprintf(`unresolved function '%s'`, n)
			})
			dh.providerFunc = func(pc lookup.ProviderContext, key string, options map[string]px.Value) (px.Value, bool) {
				return nil, false
			}
		}
	}
	return dh.providerFunc
}

func (dh *LookupKeyProvider) FullName() string {
	return fmt.Sprintf(`lookup_key function '%s'`, dh.function.Name())
}

func newLookupKeyProvider(he lookup.HierarchyEntry) lookup.DataProvider {
	ls := he.Locations()
	return &LookupKeyProvider{
		function:  he.Function(),
		locations: ls,
		hashes:    &sync.Map{},
	}
}
