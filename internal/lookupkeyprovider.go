package internal

import (
	"fmt"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/hierasdk/hiera"
)

type lookupKeyProvider struct {
	hierarchyEntry api.Entry
	providerFunc   hiera.LookupKey
}

func (dh *lookupKeyProvider) Hierarchy() api.Entry {
	return dh.hierarchyEntry
}

func (dh *lookupKeyProvider) LookupKey(key api.Key, ic api.Invocation, location api.Location) dgo.Value {
	root := key.Root()
	opts := dh.hierarchyEntry.Options()
	if location != nil {
		opts = optionsWithLocation(opts, location.Resolved())
	}
	value := dh.providerFunction(ic)(ic.ServerContext(opts), root)
	if value != nil {
		ic.ReportFound(root, value)
	} else {
		ic.ReportNotFound(root)
	}
	return value
}

func (dh *lookupKeyProvider) providerFunction(ic api.Invocation) (pf hiera.LookupKey) {
	if dh.providerFunc == nil {
		dh.providerFunc = dh.loadFunction(ic)
	}
	return dh.providerFunc
}

func (dh *lookupKeyProvider) loadFunction(ic api.Invocation) (pf hiera.LookupKey) {
	n := dh.hierarchyEntry.Function().Name()
	switch n {
	case `environment`:
		return provider.Environment
	case `scope`:
		return provider.ScopeLookupKey
	}
	if f, ok := ic.LoadFunction(dh.hierarchyEntry); ok {
		return func(pc hiera.ProviderContext, key string) dgo.Value {
			return f.Call(vf.MutableValues(pc, key))[0]
		}
	}
	ic.ReportText(func() string { return fmt.Sprintf(`unresolved function '%s'`, n) })
	return func(pc hiera.ProviderContext, key string) dgo.Value { return nil }
}

func (dh *lookupKeyProvider) FullName() string {
	return fmt.Sprintf(`lookup_key function '%s'`, dh.hierarchyEntry.Function().Name())
}

// NewLookupKeyProvider creates a new provider with a lookup_key function configured from the given entry
func NewLookupKeyProvider(he api.Entry) api.DataProvider {
	return &lookupKeyProvider{hierarchyEntry: he}
}
