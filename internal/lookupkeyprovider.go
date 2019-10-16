package internal

import (
	"fmt"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/hierasdk/hiera"
)

type LookupKeyProvider struct {
	hierarchyEntry hieraapi.Entry
	providerFunc   hiera.LookupKey
}

func (dh *LookupKeyProvider) Hierarchy() hieraapi.Entry {
	return dh.hierarchyEntry
}

func (dh *LookupKeyProvider) LookupKey(key hieraapi.Key, ic hieraapi.Invocation, location hieraapi.Location) dgo.Value {
	root := key.Root()
	opts := dh.hierarchyEntry.Options()
	if location != nil {
		opts = optionsWithLocation(opts, location.Resolved())
	}
	value := dh.providerFunction(ic)(ic.ServerContext(dh.hierarchyEntry, opts), root)
	if value != nil {
		ic.ReportFound(root, value)
	} else {
		ic.ReportNotFound(root)
	}
	return value
}

func (dh *LookupKeyProvider) providerFunction(ic hieraapi.Invocation) (pf hiera.LookupKey) {
	if dh.providerFunc == nil {
		dh.providerFunc = dh.loadFunction(ic)
	}
	return dh.providerFunc
}

func (dh *LookupKeyProvider) loadFunction(ic hieraapi.Invocation) (pf hiera.LookupKey) {
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

func (dh *LookupKeyProvider) FullName() string {
	return fmt.Sprintf(`lookup_key function '%s'`, dh.hierarchyEntry.Function().Name())
}

// NewLookupKeyProvider creates a new provider with a lookup_key function configured from the given entry
func NewLookupKeyProvider(he hieraapi.Entry) hieraapi.DataProvider {
	return &LookupKeyProvider{hierarchyEntry: he}
}
