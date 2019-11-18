package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/pcore/yaml"

	// Ensure that pcore is initialized
	_ "github.com/lyraproj/pcore/pcore"
)

type function struct {
	kind hieraapi.Kind
	name string
}

func (f *function) Kind() hieraapi.Kind {
	return f.kind
}

func (f *function) Name() string {
	return f.name
}

func (f *function) Resolve(ic hieraapi.Invocation) (hieraapi.Function, bool) {
	if n, changed := interpolateString(ic, f.name, false); changed {
		return &function{f.kind, n.String()}, true
	}
	return f, false
}

type entry struct {
	cfg        *hieraCfg
	dataDir    string
	pluginDir  string
	pluginFile string
	options    px.OrderedMap
	optsMap    map[string]px.Value
	function   hieraapi.Function
	name       string
	locations  []hieraapi.Location
}

func (e *entry) Options() px.OrderedMap {
	return e.options
}

func (e *entry) OptionsMap() map[string]px.Value {
	return e.optsMap
}

func (e *entry) DataDir() string {
	return e.dataDir
}

func (e *entry) PluginDir() string {
	return e.pluginDir
}

func (e *entry) PluginFile() string {
	return e.pluginFile
}

func (e *entry) Function() hieraapi.Function {
	return e.function
}

func (e *entry) initialize(ic hieraapi.Invocation, name string, entryHash *types.Hash) {
	entryHash.EachPair(func(k, v px.Value) {
		ks := k.String()
		if ks == `options` {
			e.options = v.(*types.Hash)
			e.options.EachKey(func(optKey px.Value) {
				if utils.ContainsString(hieraapi.ReservedOptionKeys, optKey.String()) {
					panic(px.Error(hieraapi.OptionReservedByHiera, issue.H{`key`: optKey.String(), `name`: name}))
				}
			})
		} else if utils.ContainsString(hieraapi.FunctionKeys, ks) {
			if e.function != nil {
				panic(px.Error(hieraapi.MultipleDataProviderFunctions, issue.H{`keys`: hieraapi.FunctionKeys, `name`: name}))
			}
			e.function = &function{hieraapi.Kind(ks), v.String()}
		}
	})
}

func (e *entry) Copy(cfg hieraapi.Config) hieraapi.Entry {
	c := *e
	c.cfg = cfg.(*hieraCfg)
	return &c
}

func (e *entry) Name() string {
	return e.name
}

func (e *entry) Locations() []hieraapi.Location {
	return e.locations
}

func (e *entry) CreateProvider() hieraapi.DataProvider {
	switch e.function.Kind() {
	case hieraapi.KindDataHash:
		return newDataHashProvider(e)
	case hieraapi.KindDataDig:
		return newDataDigProvider(e)
	default:
		return newLookupKeyProvider(e)
	}
}

func (e *entry) Resolve(ic hieraapi.Invocation, defaults hieraapi.Entry) hieraapi.Entry {
	// Resolve interpolated strings and locations
	ce := *e

	if ce.function == nil {
		if defaults == nil {
			ce.function = &function{kind: hieraapi.KindDataHash, name: `yaml_data`}
		} else {
			ce.function = defaults.Function()
		}
	} else if f, fc := ce.function.Resolve(ic); fc {
		ce.function = f
	}

	if ce.function == nil {
		panic(px.Error(hieraapi.MissingDataProviderFunction, issue.H{`keys`: hieraapi.FunctionKeys, `name`: e.name}))
	}

	dataDir, exists := os.LookupEnv("HIERA_DATADIR")
	if !exists {
		dataDir = `data`
	}
	pluginDir, exists := os.LookupEnv("HIERA_PLUGINDIR")
	if !exists {
		pluginDir = `plugin`
	}

	if ce.dataDir == `` {
		if defaults == nil {
			ce.dataDir = dataDir
		} else {
			ce.dataDir = defaults.DataDir()
		}
	} else {
		if d, dc := interpolateString(ic, ce.dataDir, false); dc {
			ce.dataDir = d.String()
		}
	}

	if ce.pluginDir == `` {
		if defaults == nil {
			ce.pluginDir = pluginDir
		} else {
			ce.pluginDir = defaults.PluginDir()
		}
	} else {
		if d, dc := interpolateString(ic, ce.pluginDir, false); dc {
			ce.pluginDir = d.String()
		}
	}
	if !filepath.IsAbs(ce.pluginDir) {
		ce.pluginDir = filepath.Join(e.cfg.root, ce.pluginDir)
	}

	if ce.options == nil {
		if defaults != nil {
			ce.options = defaults.Options()
			ce.optsMap = defaults.OptionsMap()
		}
	} else if ce.options.Len() > 0 {
		if o, oc := doInterpolate(ic, ce.options, false); oc {
			ce.options = o.(*types.Hash)
		}
		ce.optsMap = ce.options.ToStringMap()
	}

	var dataRoot string
	if filepath.IsAbs(ce.dataDir) {
		dataRoot = ce.dataDir
	} else {
		dataRoot = filepath.Join(e.cfg.root, ce.dataDir)
	}
	if ce.locations != nil {
		ne := make([]hieraapi.Location, 0, len(ce.locations))
		for _, l := range ce.locations {
			ne = append(ne, l.Resolve(ic, dataRoot)...)
		}
		ce.locations = ne
	}

	return &ce
}

type hieraCfg struct {
	root             string
	path             string
	defaults         *entry
	hierarchy        []hieraapi.Entry
	defaultHierarchy []hieraapi.Entry
}

func NewConfig(ic hieraapi.Invocation, configPath string) hieraapi.Config {
	b, ok := types.BinaryFromFile2(configPath)
	if !ok {
		dc := &hieraCfg{
			root:             filepath.Dir(configPath),
			path:             ``,
			defaultHierarchy: []hieraapi.Entry{},
		}
		dc.defaults = dc.makeDefaultConfig()
		dc.hierarchy = dc.makeDefaultHierarchy()
		return dc
	}

	cfgType := ic.ParseType(`Hiera::Config`)
	yv := yaml.Unmarshal(ic, b.Bytes())

	return createConfig(ic, configPath, px.AssertInstance(func() string {
		return fmt.Sprintf(`The Lookup Configuration at '%s'`, configPath)
	}, cfgType, yv).(*types.Hash))
}

func createConfig(ic hieraapi.Invocation, path string, hash *types.Hash) hieraapi.Config {
	cfg := &hieraCfg{root: filepath.Dir(path), path: path}

	if dv, ok := hash.Get4(`defaults`); ok {
		cfg.defaults = cfg.createEntry(ic, `defaults`, dv.(*types.Hash)).(*entry)
	} else {
		cfg.defaults = cfg.makeDefaultConfig()
	}

	if hv, ok := hash.Get4(`hierarchy`); ok {
		cfg.hierarchy = cfg.createHierarchy(ic, hv.(*types.Array))
	} else {
		cfg.hierarchy = cfg.makeDefaultHierarchy()
	}

	if hv, ok := hash.Get4(`default_hierarchy`); ok {
		cfg.defaultHierarchy = cfg.createHierarchy(ic, hv.(*types.Array))
	}

	return cfg
}

func (hc *hieraCfg) makeDefaultConfig() *entry {
	pluginDir, exists := os.LookupEnv("HIERA_PLUGINDIR")
	if !exists {
		pluginDir = `plugin`
	}
	dataDir, exists := os.LookupEnv("HIERA_DATADIR")
	if !exists {
		dataDir = `data`
	}
	return &entry{
		cfg:       hc,
		dataDir:   dataDir,
		pluginDir: pluginDir,
		function:  &function{kind: hieraapi.KindDataHash, name: `yaml_data`},
	}
}

func (hc *hieraCfg) makeDefaultHierarchy() []hieraapi.Entry {
	return []hieraapi.Entry{
		// The lyra default behavior is to look for a <Hiera root>/data.yaml. Hiera root is the current directory.
		&entry{cfg: hc, dataDir: `.`, name: `Root`, locations: []hieraapi.Location{&path{original: `data.yaml`}}},
		// Hiera proper default behavior is to look for <Hiera root>/data/common.yaml
		&entry{cfg: hc, name: `Common`, locations: []hieraapi.Location{&path{original: `common.yaml`}}}}
}

func (hc *hieraCfg) Resolve(ic hieraapi.Invocation) (cfg hieraapi.ResolvedConfig) {
	r := &resolvedConfig{config: hc}
	r.Resolve(ic.ForConfig())
	cfg = r

	ms := hieraapi.GetMergeStrategy(hieraapi.Deep, nil)
	k := newKey(`lookup_options`)
	ic = ic.ForLookupOptions()
	v := ic.WithLookup(k, func() px.Value {
		return ms.Lookup(r.Hierarchy(), ic, func(prv interface{}) px.Value {
			pr := prv.(hieraapi.DataProvider)
			return pr.Lookup(k, ic, ms)
		})
	})

	if lm, ok := v.(px.OrderedMap); ok {
		lo := make(map[string]map[string]px.Value, lm.Len())
		lm.EachPair(func(k, v px.Value) {
			if km, ok := v.(px.OrderedMap); ok {
				ko := make(map[string]px.Value, km.Len())
				lo[k.String()] = ko
				km.EachPair(func(k, v px.Value) {
					ko[k.String()] = v
				})
			}
		})
		r.lookupOptions = lo
	}
	return r
}

func (hc *hieraCfg) Hierarchy() []hieraapi.Entry {
	return hc.hierarchy
}

func (hc *hieraCfg) DefaultHierarchy() []hieraapi.Entry {
	return hc.defaultHierarchy
}

func (hc *hieraCfg) Root() string {
	return hc.root
}

func (hc *hieraCfg) Path() string {
	return hc.path
}

func (hc *hieraCfg) Defaults() hieraapi.Entry {
	return hc.defaults
}

func (hc *hieraCfg) CreateProviders(ic hieraapi.Invocation, hierarchy []hieraapi.Entry) []hieraapi.DataProvider {
	providers := make([]hieraapi.DataProvider, len(hierarchy))
	defaults := hc.defaults.Resolve(ic, nil)
	for i, he := range hierarchy {
		providers[i] = he.Resolve(ic, defaults).CreateProvider()
	}
	return providers
}

func (hc *hieraCfg) createHierarchy(ic hieraapi.Invocation, hierarchy *types.Array) []hieraapi.Entry {
	entries := make([]hieraapi.Entry, 0, hierarchy.Len())
	uniqueNames := make(map[string]bool, hierarchy.Len())
	hierarchy.Each(func(hv px.Value) {
		hh := hv.(*types.Hash)
		name := hh.Get5(`name`, px.EmptyString).String()
		if uniqueNames[name] {
			panic(px.Error(hieraapi.HierarchyNameMultiplyDefined, issue.H{`name`: name}))
		}
		uniqueNames[name] = true
		entries = append(entries, hc.createEntry(ic, name, hh))
	})
	return entries
}

func (hc *hieraCfg) createEntry(ic hieraapi.Invocation, name string, entryHash *types.Hash) hieraapi.Entry {
	entry := &entry{cfg: hc, name: name}
	entry.initialize(ic, name, entryHash)
	entryHash.EachPair(func(k, v px.Value) {
		ks := k.String()
		switch {
		case ks == `datadir`:
			entry.dataDir = v.String()
		case ks == `plugindir`:
			entry.pluginDir = v.String()
		case ks == `pluginfile`:
			entry.pluginFile = v.String()
		case utils.ContainsString(hieraapi.LocationKeys, ks):
			if entry.locations != nil {
				panic(px.Error(hieraapi.MultipleLocationSpecs, issue.H{`keys`: hieraapi.LocationKeys, `name`: name}))
			}
			switch ks {
			case `path`:
				entry.locations = []hieraapi.Location{&path{original: v.String()}}
			case `paths`:
				a := v.(*types.Array)
				entry.locations = make([]hieraapi.Location, 0, a.Len())
				a.Each(func(p px.Value) { entry.locations = append(entry.locations, &path{original: p.String()}) })
			case `glob`:
				entry.locations = []hieraapi.Location{&glob{v.String()}}
			case `globs`:
				a := v.(*types.Array)
				entry.locations = make([]hieraapi.Location, 0, a.Len())
				a.Each(func(p px.Value) { entry.locations = append(entry.locations, &glob{p.String()}) })
			case `uri`:
				entry.locations = []hieraapi.Location{&uri{original: v.String()}}
			case `uris`:
				a := v.(*types.Array)
				entry.locations = make([]hieraapi.Location, 0, a.Len())
				a.Each(func(p px.Value) { entry.locations = append(entry.locations, &uri{original: p.String()}) })
			default: // Mapped paths
				a := v.(*types.Array)
				entry.locations = []hieraapi.Location{&mappedPaths{a.At(0).String(), a.At(1).String(), a.At(2).String()}}
			}
		}
	})
	return entry
}

type resolvedConfig struct {
	config           *hieraCfg
	providers        []hieraapi.DataProvider
	defaultProviders []hieraapi.DataProvider
	lookupOptions    map[string]map[string]px.Value
}

func (r *resolvedConfig) Config() hieraapi.Config {
	return r.config
}

func (r *resolvedConfig) Hierarchy() []hieraapi.DataProvider {
	return r.providers
}

func (r *resolvedConfig) DefaultHierarchy() []hieraapi.DataProvider {
	return r.defaultProviders
}

func (r *resolvedConfig) LookupOptions(key hieraapi.Key) map[string]px.Value {
	if r.lookupOptions != nil {
		return r.lookupOptions[key.Root()]
	}
	return nil
}

func (r *resolvedConfig) Resolve(ic hieraapi.Invocation) {
	r.providers = r.config.CreateProviders(ic, r.config.Hierarchy())
	r.defaultProviders = r.config.CreateProviders(ic, r.config.DefaultHierarchy())
}
