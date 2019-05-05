package impl

import (
	"fmt"
	"path/filepath"

	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/pcore/yaml"

	// Ensure that pcore is initialized
	_ "github.com/lyraproj/pcore/pcore"
)

type function struct {
	kind lookup.LookupKind
	name string
}

func (f *function) Kind() lookup.LookupKind {
	return f.kind
}

func (f *function) Name() string {
	return f.name
}

func (f *function) Resolve(ic lookup.Invocation) (lookup.Function, bool) {
	if n, changed := interpolateString(ic, f.name, false); changed {
		return &function{f.kind, n.String()}, true
	}
	return f, false
}

type entry struct {
	cfg      *hieraCfg
	dataDir  string
	options  px.OrderedMap
	function lookup.Function
}

func (e *entry) Options() px.OrderedMap {
	return e.options
}

func (e *entry) DataDir() string {
	return e.dataDir
}

func (e *entry) Function() lookup.Function {
	return e.function
}

func (e *entry) initialize(ic lookup.Invocation, name string, entryHash *types.Hash) {
	entryHash.EachPair(func(k, v px.Value) {
		ks := k.String()
		if ks == `options` {
			e.options = v.(*types.Hash)
			e.options.EachKey(func(optKey px.Value) {
				if utils.ContainsString(lookup.ReservedOptionKeys, optKey.String()) {
					panic(px.Error(OptionReservedByHiera, issue.H{`key`: optKey.String(), `name`: name}))
				}
			})
		} else if utils.ContainsString(lookup.FunctionKeys, ks) {
			if e.function != nil {
				panic(px.Error(MultipleDataProviderFunctions, issue.H{`keys`: lookup.FunctionKeys, `name`: name}))
			}
			e.function = &function{lookup.LookupKind(ks), v.String()}
		}
	})
}

func (e *entry) Copy(cfg lookup.Config) lookup.Entry {
	c := *e
	c.cfg = cfg.(*hieraCfg)
	return &c
}

type hierEntry struct {
	entry
	name      string
	locations []lookup.Location
}

func (e *hierEntry) Copy(cfg lookup.Config) lookup.Entry {
	c := *e
	c.cfg = cfg.(*hieraCfg)
	return &c
}

func (e *hierEntry) Name() string {
	return e.name
}

func (e *hierEntry) Locations() []lookup.Location {
	return e.locations
}

func (e *hierEntry) CreateProvider() lookup.DataProvider {
	switch e.function.Kind() {
	case lookup.KindDataHash:
		return newDataHashProvider(e)
	case lookup.KindDataDig:
		return newDataDigProvider(e)
	default:
		return newLookupKeyProvider(e)
	}
}

func (e *hierEntry) Resolve(ic lookup.Invocation, defaults lookup.Entry) lookup.HierarchyEntry {
	// Resolve interpolated strings and locations
	ce := *e

	if ce.function == nil {
		ce.function = defaults.Function()
	} else if f, fc := ce.function.Resolve(ic); fc {
		ce.function = f
	}

	if ce.function == nil {
		panic(px.Error(MissingDataProviderFunction, issue.H{`keys`: lookup.FunctionKeys, `name`: e.name}))
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
		ne := make([]lookup.Location, 0, len(ce.locations))
		for _, l := range ce.locations {
			ne = append(ne, l.Resolve(ic, dataRoot)...)
		}
		ce.locations = ne
	}

	return &ce
}

func init() {
	px.RegisterResolvableType(px.NewNamedType(`Hiera`, `TypeSet[{
		pcore_version => '1.0.0',
		version => '5.0.0',
		types => {
			Options => Hash[Pattern[/\A[A-Za-z](:?[0-9A-Za-z_-]*[0-9A-Za-z])?\z/], Data],
			Defaults => Struct[{
				Optional[options] => Options,
				Optional[data_dig] => String[1],
				Optional[data_hash] => String[1],
				Optional[lookup_key] => String[1],
				Optional[data_dir] => String[1],
			}],
			Entry => Struct[{
				name => String[1],
				Optional[options] => Options,
				Optional[data_dig] => String[1],
				Optional[data_hash] => String[1],
				Optional[lookup_key] => String[1],
				Optional[data_dir] => String[1],
				Optional[path] => String[1],
				Optional[paths] => Array[String[1], 1],
				Optional[glob] => String[1],
				Optional[globs] => Array[String[1], 1],
				Optional[uri] => String[1],
				Optional[uris] => Array[String[1], 1],
				Optional[mapped_paths] => Array[String[1], 3, 3],
			}],
			Config => Struct[{
				version => Integer[5, 5],
				Optional[defaults] => Defaults,
				Optional[hierarchy] => Array[Entry],
				Optional[default_hierarchy] => Array[Entry]
			}]
		}
  }]`).(px.ResolvableType))
	pcore.DefineSetting(`hiera_config`, types.DefaultStringType(), nil)

	lookup.NotFound = px.Error(KeyNotFound, issue.NoArgs)
}

type hieraCfg struct {
	root             string
	path             string
	loadedHash       px.OrderedMap
	defaults         lookup.Entry
	hierarchy        []lookup.HierarchyEntry
	defaultHierarchy []lookup.HierarchyEntry
}

func NewConfig(ic lookup.Invocation, configPath string) lookup.Config {
	b, ok := types.BinaryFromFile2(configPath)
	if !ok {
		dc := &hieraCfg{
			root:             filepath.Dir(configPath),
			path:             ``,
			loadedHash:       nil,
			defaultHierarchy: []lookup.HierarchyEntry{},
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

func createConfig(ic lookup.Invocation, path string, hash *types.Hash) lookup.Config {
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
	return &entry{cfg: hc, dataDir: `data`, function: &function{kind: lookup.KindDataHash, name: `yaml_data`}}
}

func (hc *hieraCfg) makeDefaultHierarchy() []lookup.HierarchyEntry {
	return []lookup.HierarchyEntry{
		// The lyra default behavior is to look for a <Hiera root>/data.yaml. Hiera root is the current directory.
		&hierEntry{entry: entry{cfg: hc, dataDir: `.`}, name: `Root`, locations: []lookup.Location{&path{original: `data.yaml`}}},
		// Hiera proper default behavior is to look for <Hiera root>/data/common.yaml
		&hierEntry{entry: entry{cfg: hc}, name: `Common`, locations: []lookup.Location{&path{original: `common.yaml`}}}}
}

func (hc *hieraCfg) Resolve(ic lookup.Invocation) (cfg lookup.ResolvedConfig) {
	r := &resolvedConfig{config: hc}
	r.Resolve(ic)
	cfg = r

	defer func() {
		if r := recover(); r != nil {
			// lookup.NotFound is ok. It just means that there was no lookup_options
			if r != lookup.NotFound {
				panic(r)
			}
		}
	}()

	ms := lookup.GetMergeStrategy(`deep`, nil)
	k := NewKey(`lookup_options`)
	v := ms.Lookup(r.Hierarchy(), ic, func(prv interface{}) px.Value {
		pr := prv.(lookup.DataProvider)
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

func (hc *hieraCfg) Hierarchy() []lookup.HierarchyEntry {
	return hc.hierarchy
}

func (hc *hieraCfg) DefaultHierarchy() []lookup.HierarchyEntry {
	return hc.defaultHierarchy
}

func (hc *hieraCfg) Root() string {
	return hc.root
}

func (hc *hieraCfg) Path() string {
	return hc.path
}

func (hc *hieraCfg) LoadedConfig() px.OrderedMap {
	return hc.loadedHash
}

func (hc *hieraCfg) Defaults() lookup.Entry {
	return hc.defaults
}

func (hc *hieraCfg) CreateProviders(ic lookup.Invocation, hierarchy []lookup.HierarchyEntry) []lookup.DataProvider {
	providers := make([]lookup.DataProvider, len(hierarchy))
	var defaults lookup.Entry
	if hdf, ok := hc.defaults.(*hierEntry); ok {
		defaults = hdf.Resolve(ic, nil)
	} else {
		defaults = hc.defaults.Copy(hc)
	}
	for i, he := range hierarchy {
		providers[i] = he.(*hierEntry).Resolve(ic, defaults).CreateProvider()
	}
	return providers
}

func (hc *hieraCfg) createHierarchy(ic lookup.Invocation, hier *types.Array) []lookup.HierarchyEntry {
	entries := make([]lookup.HierarchyEntry, 0, hier.Len())
	uniqueNames := make(map[string]bool, hier.Len())
	hier.Each(func(hv px.Value) {
		hh := hv.(*types.Hash)
		name := hh.Get5(`name`, px.EmptyString).String()
		if uniqueNames[name] {
			panic(px.Error(HierarchyNameMultiplyDefined, issue.H{`name`: name}))
		}
		uniqueNames[name] = true
		entries = append(entries, hc.createHierarchyEntry(ic, name, hh))
	})
	return entries
}

func (hc *hieraCfg) createDefaultsEntry(ic lookup.Invocation, entryHash *types.Hash) lookup.Entry {
	defaults := &entry{cfg: hc}
	defaults.initialize(ic, `defaults`, entryHash)
	return defaults
}

func (hc *hieraCfg) createHierarchyEntry(ic lookup.Invocation, name string, entryHash *types.Hash) lookup.HierarchyEntry {
	entry := &hierEntry{entry: entry{cfg: hc}, name: name}
	entry.initialize(ic, name, entryHash)
	entryHash.EachPair(func(k, v px.Value) {
		ks := k.String()
		if ks == `data_dir` {
			entry.dataDir = v.String()
		}
		if utils.ContainsString(lookup.LocationKeys, ks) {
			if entry.locations != nil {
				panic(px.Error(MultipleLocationSpecs, issue.H{`keys`: lookup.LocationKeys, `name`: name}))
			}
			switch ks {
			case `path`:
				entry.locations = []lookup.Location{&path{original: v.String()}}
			case `paths`:
				a := v.(*types.Array)
				entry.locations = make([]lookup.Location, 0, a.Len())
				a.Each(func(p px.Value) { entry.locations = append(entry.locations, &path{original: p.String()}) })
			case `glob`:
				entry.locations = []lookup.Location{&glob{v.String()}}
			case `globs`:
				a := v.(*types.Array)
				entry.locations = make([]lookup.Location, 0, a.Len())
				a.Each(func(p px.Value) { entry.locations = append(entry.locations, &glob{p.String()}) })
			case `uri`:
				entry.locations = []lookup.Location{&uri{original: v.String()}}
			case `uris`:
				a := v.(*types.Array)
				entry.locations = make([]lookup.Location, 0, a.Len())
				a.Each(func(p px.Value) { entry.locations = append(entry.locations, &uri{original: p.String()}) })
			default: // Mapped paths
				a := v.(*types.Array)
				entry.locations = []lookup.Location{&mappedPaths{a.At(0).String(), a.At(1).String(), a.At(2).String()}}
			}
		}
	})
	return entry
}

type resolvedConfig struct {
	config           *hieraCfg
	providers        []lookup.DataProvider
	defaultProviders []lookup.DataProvider
	lookupOptions    map[string]map[string]px.Value
}

func (r *resolvedConfig) Config() lookup.Config {
	return r.config
}

func (r *resolvedConfig) Hierarchy() []lookup.DataProvider {
	return r.providers
}

func (r *resolvedConfig) DefaultHierarchy() []lookup.DataProvider {
	return r.defaultProviders
}

func (r *resolvedConfig) LookupOptions(key lookup.Key) map[string]px.Value {
	if r.lookupOptions != nil {
		return r.lookupOptions[key.Root()]
	}
	return nil
}

func (r *resolvedConfig) Resolve(ic lookup.Invocation) {
	r.providers = r.config.CreateProviders(ic, r.config.Hierarchy())
	r.defaultProviders = r.config.CreateProviders(ic, r.config.DefaultHierarchy())
}
