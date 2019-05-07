package internal

import (
	"fmt"
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
	cfg      *hieraCfg
	dataDir  string
	options  px.OrderedMap
	function hieraapi.Function
}

func (e *entry) Options() px.OrderedMap {
	return e.options
}

func (e *entry) DataDir() string {
	return e.dataDir
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

type hieraEntry struct {
	entry
	name      string
	locations []hieraapi.Location
}

func (e *hieraEntry) Copy(cfg hieraapi.Config) hieraapi.Entry {
	c := *e
	c.cfg = cfg.(*hieraCfg)
	return &c
}

func (e *hieraEntry) Name() string {
	return e.name
}

func (e *hieraEntry) Locations() []hieraapi.Location {
	return e.locations
}

func (e *hieraEntry) CreateProvider() hieraapi.DataProvider {
	switch e.function.Kind() {
	case hieraapi.KindDataHash:
		return newDataHashProvider(e)
	case hieraapi.KindDataDig:
		return newDataDigProvider(e)
	default:
		return newLookupKeyProvider(e)
	}
}

func (e *hieraEntry) Resolve(ic hieraapi.Invocation, defaults hieraapi.Entry) hieraapi.HierarchyEntry {
	// Resolve interpolated strings and locations
	ce := *e

	if ce.function == nil {
		ce.function = defaults.Function()
	} else if f, fc := ce.function.Resolve(ic); fc {
		ce.function = f
	}

	if ce.function == nil {
		panic(px.Error(hieraapi.MissingDataProviderFunction, issue.H{`keys`: hieraapi.FunctionKeys, `name`: e.name}))
	}

	if ce.dataDir == `` {
		ce.dataDir = defaults.DataDir()
	} else {
		if d, dc := interpolateString(ic, ce.dataDir, false); dc {
			ce.dataDir = d.String()
		}
	}

	if ce.options == nil {
		ce.options = defaults.Options()
	} else if ce.options.Len() > 0 {
		if o, oc := doInterpolate(ic, ce.options, false); oc {
			ce.options = o.(*types.Hash)
		}
	}

	dataRoot := filepath.Join(e.cfg.root, ce.dataDir)
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
	defaults         hieraapi.Entry
	hierarchy        []hieraapi.HierarchyEntry
	defaultHierarchy []hieraapi.HierarchyEntry
}

func NewConfig(ic hieraapi.Invocation, configPath string) hieraapi.Config {
	b, ok := types.BinaryFromFile2(configPath)
	if !ok {
		dc := &hieraCfg{
			root:             filepath.Dir(configPath),
			path:             ``,
			defaultHierarchy: []hieraapi.HierarchyEntry{},
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
		cfg.defaults = cfg.createDefaultsEntry(ic, dv.(*types.Hash))
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
	return &entry{cfg: hc, dataDir: `data`, function: &function{kind: hieraapi.KindDataHash, name: `yaml_data`}}
}

func (hc *hieraCfg) makeDefaultHierarchy() []hieraapi.HierarchyEntry {
	return []hieraapi.HierarchyEntry{
		// The lyra default behavior is to look for a <Hiera root>/data.yaml. Hiera root is the current directory.
		&hieraEntry{entry: entry{cfg: hc, dataDir: `.`}, name: `Root`, locations: []hieraapi.Location{&path{original: `data.yaml`}}},
		// Hiera proper default behavior is to look for <Hiera root>/data/common.yaml
		&hieraEntry{entry: entry{cfg: hc}, name: `Common`, locations: []hieraapi.Location{&path{original: `common.yaml`}}}}
}

func (hc *hieraCfg) Resolve(ic hieraapi.Invocation) (cfg hieraapi.ResolvedConfig) {
	r := &resolvedConfig{config: hc}
	r.Resolve(ic)
	cfg = r

	ms := hieraapi.GetMergeStrategy(`deep`, nil)
	k := newKey(`lookup_options`)
	v := ms.Lookup(r.Hierarchy(), ic, func(prv interface{}) px.Value {
		pr := prv.(hieraapi.DataProvider)
		return pr.UncheckedLookup(k, ic, ms)
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

func (hc *hieraCfg) Hierarchy() []hieraapi.HierarchyEntry {
	return hc.hierarchy
}

func (hc *hieraCfg) DefaultHierarchy() []hieraapi.HierarchyEntry {
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

func (hc *hieraCfg) CreateProviders(ic hieraapi.Invocation, hierarchy []hieraapi.HierarchyEntry) []hieraapi.DataProvider {
	providers := make([]hieraapi.DataProvider, len(hierarchy))
	var defaults hieraapi.Entry
	if hdf, ok := hc.defaults.(*hieraEntry); ok {
		defaults = hdf.Resolve(ic, nil)
	} else {
		defaults = hc.defaults.Copy(hc)
	}
	for i, he := range hierarchy {
		providers[i] = he.(*hieraEntry).Resolve(ic, defaults).CreateProvider()
	}
	return providers
}

func (hc *hieraCfg) createHierarchy(ic hieraapi.Invocation, hierarchy *types.Array) []hieraapi.HierarchyEntry {
	entries := make([]hieraapi.HierarchyEntry, 0, hierarchy.Len())
	uniqueNames := make(map[string]bool, hierarchy.Len())
	hierarchy.Each(func(hv px.Value) {
		hh := hv.(*types.Hash)
		name := hh.Get5(`name`, px.EmptyString).String()
		if uniqueNames[name] {
			panic(px.Error(hieraapi.HierarchyNameMultiplyDefined, issue.H{`name`: name}))
		}
		uniqueNames[name] = true
		entries = append(entries, hc.createHierarchyEntry(ic, name, hh))
	})
	return entries
}

func (hc *hieraCfg) createDefaultsEntry(ic hieraapi.Invocation, entryHash *types.Hash) hieraapi.Entry {
	defaults := &entry{cfg: hc}
	defaults.initialize(ic, `defaults`, entryHash)
	return defaults
}

func (hc *hieraCfg) createHierarchyEntry(ic hieraapi.Invocation, name string, entryHash *types.Hash) hieraapi.HierarchyEntry {
	entry := &hieraEntry{entry: entry{cfg: hc}, name: name}
	entry.initialize(ic, name, entryHash)
	entryHash.EachPair(func(k, v px.Value) {
		ks := k.String()
		if ks == `data_dir` {
			entry.dataDir = v.String()
		}
		if utils.ContainsString(hieraapi.LocationKeys, ks) {
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
