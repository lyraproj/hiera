package session

import (
	"strings"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/internal"
	"github.com/lyraproj/hiera/merge"
)

type (
	resolvedConfig struct {
		cfg              api.Config
		providers        []api.DataProvider
		defaultProviders []api.DataProvider
		lookupOptions    dgo.Map
		moduleName       string
	}
)

// CreateProvider creates and returns the DataProvider configured by the given entry
func CreateProvider(e api.Entry) api.DataProvider {
	switch e.Function().Kind() {
	case api.KindDataHash:
		return internal.NewDataHashProvider(e)
	case api.KindDataDig:
		return internal.NewDataDigProvider(e)
	default:
		return internal.NewLookupKeyProvider(e)
	}
}

// Resolve resolves the given Config into a ResolvedConfig. Resolving means creating the proper
// DataProviders for all Hierarchy entries
func Resolve(ic api.Invocation, hc api.Config, moduleName string) api.ResolvedConfig {
	r := &resolvedConfig{cfg: hc, moduleName: moduleName}
	r.Resolve(ic)
	return r
}

func (r *resolvedConfig) Config() api.Config {
	return r.cfg
}

func (r *resolvedConfig) Hierarchy() []api.DataProvider {
	return r.providers
}

func (r *resolvedConfig) DefaultHierarchy() []api.DataProvider {
	return r.defaultProviders
}

func (r *resolvedConfig) LookupOptions(key api.Key) dgo.Map {
	root := key.Root()
	if r.lookupOptions != nil && (r.moduleName == `` || strings.HasPrefix(root, r.moduleName+`::`)) {
		if m, ok := r.lookupOptions.Get(root).(dgo.Map); ok {
			return m
		}
	}
	return nil
}

func (r *resolvedConfig) Resolve(ic api.Invocation) {
	icc := ic.ForConfig()
	r.providers = r.CreateProviders(icc, r.cfg.Hierarchy())
	r.defaultProviders = r.CreateProviders(icc, r.cfg.DefaultHierarchy())

	ms := merge.GetStrategy(`deep`, nil)
	k := api.NewKey(`lookup_options`)
	lic := ic.ForLookupOptions()
	v := lic.WithLookup(k, func() dgo.Value {
		return ms.MergeLookup(r.Hierarchy(), lic, func(prv interface{}) dgo.Value {
			pr := prv.(api.DataProvider)
			return lic.MergeLocations(k, pr, ms)
		})
	})

	if lm, ok := v.(dgo.Map); ok {
		r.lookupOptions = lm
	}
}

func (r *resolvedConfig) CreateProviders(ic api.Invocation, hierarchy []api.Entry) []api.DataProvider {
	providers := make([]api.DataProvider, len(hierarchy))
	defaults := r.cfg.Defaults().Resolve(ic, nil)
	for i, he := range hierarchy {
		providers[i] = CreateProvider(he.Resolve(ic, defaults))
	}
	return providers
}
