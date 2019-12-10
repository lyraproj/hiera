package provider

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
)

// ConfigLookupKey performs a lookup based on a hierarchy of providers that has been specified
// in a yaml based configuration stored on disk.
func ConfigLookupKey(pc hiera.ProviderContext, key string) dgo.Value {
	if sc, ok := pc.(hieraapi.ServerContext); ok {
		return ConfigLookupKeyAt(sc, sc.Invocation().SessionOptions().Get(hieraapi.HieraConfig).String(), key, ``)
	}
	return nil
}

// ConfigLookupKeyAt performs a lookup based on a hierarchy of providers that has been specified
// in a yaml based configuration appointed by the given configPath.
func ConfigLookupKeyAt(sc hieraapi.ServerContext, configPath, key, moduleName string) dgo.Value {
	ic := sc.Invocation()
	cfg := ic.Config(configPath, moduleName)
	k := hieraapi.NewKey(key)
	if ic.LookupOptionsMode() {
		return cfg.LookupOptions(k)
	}

	if ic.DataMode() {
		return ic.MergeHierarchy(k, cfg.Hierarchy(), ic.MergeStrategy())
	}

	ic = sc.Invocation().ForData()
	return ic.WithLookup(k, func() dgo.Value {
		ic.SetMergeStrategy(sc.Option(`merge`), cfg.LookupOptions(k))
		return ic.LookupAndConvertData(func() dgo.Value {
			return ic.MergeHierarchy(k, cfg.Hierarchy(), ic.MergeStrategy())
		})
	})
}
