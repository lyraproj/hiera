package session

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/internal"
	"github.com/lyraproj/hiera/merge"
)

type (
	resolvedConfig struct {
		cfg              hieraapi.Config
		providers        []hieraapi.DataProvider
		defaultProviders []hieraapi.DataProvider
		lookupOptions    dgo.Map
	}
)

// CreateProvider creates and returns the DataProvider configured by the given entry
func CreateProvider(e hieraapi.Entry) hieraapi.DataProvider {
	switch e.Function().Kind() {
	case hieraapi.KindDataHash:
		return internal.NewDataHashProvider(e)
	case hieraapi.KindDataDig:
		return internal.NewDataDigProvider(e)
	default:
		return internal.NewLookupKeyProvider(e)
	}
}

// Resolve resolves the given Config into a ResolvedConfig. Resolving means creating the proper
// DataProviders for all Hierarchy entries
func Resolve(ic hieraapi.Invocation, hc hieraapi.Config) hieraapi.ResolvedConfig {
	r := &resolvedConfig{cfg: hc}
	r.Resolve(ic)
	return r
}

func (r *resolvedConfig) Config() hieraapi.Config {
	return r.cfg
}

func (r *resolvedConfig) Hierarchy() []hieraapi.DataProvider {
	return r.providers
}

func (r *resolvedConfig) DefaultHierarchy() []hieraapi.DataProvider {
	return r.defaultProviders
}

func (r *resolvedConfig) LookupOptions(key hieraapi.Key) dgo.Map {
	if r.lookupOptions != nil {
		if m, ok := r.lookupOptions.Get(key.Root()).(dgo.Map); ok {
			return m
		}
	}
	return nil
}

func (r *resolvedConfig) Resolve(ic hieraapi.Invocation) {
	icc := ic.ForConfig()
	r.providers = r.CreateProviders(icc, r.cfg.Hierarchy())
	r.defaultProviders = r.CreateProviders(icc, r.cfg.DefaultHierarchy())

	ms := merge.GetStrategy(`deep`, nil)
	k := hieraapi.NewKey(`lookup_options`)
	lic := ic.ForLookupOptions()
	v := lic.WithLookup(k, func() dgo.Value {
		return ms.MergeLookup(r.Hierarchy(), lic, func(prv interface{}) dgo.Value {
			pr := prv.(hieraapi.DataProvider)
			return lic.MergeLookup(k, pr, ms)
		})
	})

	if lm, ok := v.(dgo.Map); ok {
		r.lookupOptions = lm
	}
}

func (r *resolvedConfig) CreateProviders(ic hieraapi.Invocation, hierarchy []hieraapi.Entry) []hieraapi.DataProvider {
	providers := make([]hieraapi.DataProvider, len(hierarchy))
	defaults := r.cfg.Defaults().Resolve(ic, nil)
	for i, he := range hierarchy {
		providers[i] = CreateProvider(he.Resolve(ic, defaults))
	}
	return providers
}
