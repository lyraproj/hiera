package impl

import (
	"fmt"
	"path/filepath"

	"github.com/lyraproj/hiera/config"
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
	kind config.LookupKind
	name string
}

func (f *function) Kind() config.LookupKind {
	return f.kind
}

func (f *function) Name() string {
	return f.name
}

func (f *function) Resolve(ic lookup.Invocation) (config.Function, bool) {
	if n, changed := interpolateString(ic, f.name, false); changed {
		return &function{f.kind, n.String()}, true
	}
	return f, false
}

type entry struct {
	dataDir  string
	options  px.OrderedMap
	function config.Function
}

func (e *entry) Options() px.OrderedMap {
	return e.options
}

func (e *entry) DataDir() string {
	return e.dataDir
}

func (e *entry) Function() config.Function {
	return e.function
}

type hierEntry struct {
	entry
	name      string
	locations []lookup.Location
}

func (e *hierEntry) Name() string {
	return e.name
}

func (e *hierEntry) CreateProvider(ic lookup.Invocation) lookup.DataProvider {
	switch e.function.Kind() {
	case config.DataHash:
		return newDataHashProvider(ic, e)
	case config.DataDig:
		return newDataDigProvider(ic, e)
	default:
		return newLookupKeyProvider(ic, e)
	}
}

func (e *hierEntry) Resolve(ic lookup.Invocation, defaults config.Entry) config.HierarchyEntry {
	// Resolve interpolated strings and locations
	ce := *e

	if e.function == nil {
		e.function = defaults.Function()
	} else if f, fc := e.function.Resolve(ic); fc {
		ce.function = f
	}

	if e.function == nil {
		panic(px.Error(HieraMissingDataProviderFunction, issue.H{`keys`: config.FunctionKeys, `name`: e.name}))
	}

	if e.dataDir == `` {
		ce.dataDir = defaults.DataDir()
	} else {
		if d, dc := interpolateString(ic, e.dataDir, false); dc {
			ce.dataDir = d.String()
		}
	}

	if e.options == nil {
		e.options = defaults.Options()
	} else if e.options.Len() > 0 {
		if o, oc := doInterpolate(ic, e.options, false); oc {
			ce.options = o.(*types.Hash)
		}
	}

	if e.locations != nil {
		ne := make([]lookup.Location, 0, len(e.locations))
		for _, l := range e.locations {
			ne = append(ne, l.Resolve(ic, ce.dataDir)...)
		}
		ce.locations = ne
	}

	return &ce
}

var hieraTypeSet px.Type

var DefaultConfig config.Config

func init() {
	hieraTypeSet = px.NewNamedType(`Hiera`, `TypeSet{
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
  }`)

	DefaultConfig = &hieraCfg{
		root:             ``,
		path:             ``,
		loadedHash:       nil,
		defaults:         &entry{dataDir: `data`, function: &function{kind: config.DataHash, name: `yaml_data`}},
		hierarchy:        []config.HierarchyEntry{&hierEntry{name: `Common`, locations: []lookup.Location{&path{original: `common.yaml`}}}},
		defaultHierarchy: []config.HierarchyEntry{},
	}

	pcore.DefineSetting(`hiera_config`, types.DefaultStringType(), nil)
}

type hieraCfg struct {
	root             string
	path             string
	loadedHash       px.OrderedMap
	defaults         config.Entry
	hierarchy        []config.HierarchyEntry
	defaultHierarchy []config.HierarchyEntry
}

func NewConfig(ic lookup.Invocation, configPath string) config.Config {

	// TODO: Cache parsed file content
	if b, ok := types.BinaryFromFile2(configPath); ok {
		v, ok := px.Load(ic, px.NewTypedName(px.NsType, `Hiera::Config`))
		if !ok {
			panic(px.Error(px.Failure, issue.H{`message`: `Unable to load Hiera::Config data type`}))
		}
		cfgType := v.(px.Type)
		yv := yaml.Unmarshal(ic, b.Bytes())
		return createConfig(ic, configPath, px.AssertInstance(func() string {
			return fmt.Sprintf(`The Lookup Configuration at '%s'`, configPath)
		}, cfgType, yv).(*types.Hash))
	}
	return DefaultConfig
}

func (hc *hieraCfg) Resolve(ic lookup.Invocation) config.ResolvedConfig {
	r := &resolvedConfig{config: hc}
	return r.ReResolve(ic)
}

func (hc *hieraCfg) Hierarchy() []config.HierarchyEntry {
	return hc.hierarchy
}

func (hc *hieraCfg) DefaultHierarchy() []config.HierarchyEntry {
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

func (hc *hieraCfg) Defaults() config.Entry {
	return hc.defaults
}

func (hc *hieraCfg) CreateProviders(ic lookup.Invocation, hierarchy []config.HierarchyEntry) []lookup.DataProvider {
	providers := make([]lookup.DataProvider, len(hierarchy))
	defaults := hc.defaults.(*hierEntry).Resolve(ic, nil)
	for i, he := range hierarchy {
		providers[i] = he.(*hierEntry).Resolve(ic, defaults).CreateProvider(ic)
	}
	return providers
}

func createConfig(ic lookup.Invocation, path string, hash *types.Hash) config.Config {
	cfg := &hieraCfg{root: filepath.Dir(path), path: path}

	if dv, ok := hash.Get4(`defaults`); ok {
		cfg.defaults = createDefaultsEntry(ic, dv.(*types.Hash))
	} else {
		cfg.defaults = DefaultConfig.Defaults()
	}

	if hv, ok := hash.Get4(`hierarchy`); ok {
		cfg.hierarchy = createHierarchy(ic, hv.(*types.Array))
	} else {
		cfg.hierarchy = DefaultConfig.Hierarchy()
	}

	if hv, ok := hash.Get4(`default_hierarchy`); ok {
		cfg.defaultHierarchy = createHierarchy(ic, hv.(*types.Array))
	}

	return cfg
}

func createHierarchy(ic lookup.Invocation, hier *types.Array) []config.HierarchyEntry {
	entries := make([]config.HierarchyEntry, 0, hier.Len())
	uniqueNames := make(map[string]bool, hier.Len())
	hier.Each(func(hv px.Value) {
		hh := hv.(*types.Hash)
		name := hh.Get5(`name`, px.EmptyString).String()
		if uniqueNames[name] {
			panic(px.Error(HieraHierarchyNameMultiplyDefined, issue.H{`name`: name}))
		}
		uniqueNames[name] = true
		entries = append(entries, createHierarchyEntry(ic, name, hh))
	})
	return entries
}

func (e *entry) initialize(ic lookup.Invocation, name string, entryHash *types.Hash) {
	entryHash.EachPair(func(k, v px.Value) {
		ks := k.String()
		if ks == `options` {
			e.options = v.(*types.Hash)
			e.options.EachKey(func(optKey px.Value) {
				if utils.ContainsString(config.ReservedOptionKeys, optKey.String()) {
					panic(px.Error(HieraOptionReservedByPuppet, issue.H{`key`: optKey.String(), `name`: name}))
				}
			})
		} else if utils.ContainsString(config.FunctionKeys, ks) {
			if e.function != nil {
				panic(px.Error(HieraMultipleDataProviderFunctions, issue.H{`keys`: config.FunctionKeys, `name`: name}))
			}
			e.function = &function{config.LookupKind(ks), v.String()}
		}
	})
}

func createDefaultsEntry(ic lookup.Invocation, entryHash *types.Hash) config.Entry {
	defaults := &entry{}
	defaults.initialize(ic, `defaults`, entryHash)
	return defaults
}

func createHierarchyEntry(ic lookup.Invocation, name string, entryHash *types.Hash) config.HierarchyEntry {
	entry := &hierEntry{name: name}
	entry.initialize(ic, name, entryHash)
	entryHash.EachPair(func(k, v px.Value) {
		ks := k.String()
		if utils.ContainsString(config.LocationKeys, ks) {
			if entry.locations != nil {
				panic(px.Error(HieraMultipleLocationSpecs, issue.H{`keys`: config.LocationKeys, `name`: name}))
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
	variablesUsed    map[string]px.Value
	providers        []lookup.DataProvider
	defaultProviders []lookup.DataProvider
}

func (r *resolvedConfig) ReResolve(ic lookup.Invocation) config.ResolvedConfig {
	if r.variablesUsed == nil {
		r.Resolve(ic)
		return r
	}

	allEqual := true
	scope := ic.Scope()
	for k, v := range r.variablesUsed {
		if sv, ok := scope.Get(types.WrapString(k)); ok && v.Equals(sv, nil) {
			continue
		}
		allEqual = false
		break
	}
	if allEqual {
		return r
	}

	rr := &resolvedConfig{config: r.config}
	rr.Resolve(ic)
	return rr
}

func (r *resolvedConfig) Config() config.Config {
	return r.config
}

func (r *resolvedConfig) Hierarchy() []lookup.DataProvider {
	return r.providers
}

func (r *resolvedConfig) DefaultHierarchy() []lookup.DataProvider {
	return r.defaultProviders
}

func (r *resolvedConfig) Resolve(ic lookup.Invocation) {
	ts := NewTrackingScope(ic.Scope())
	ic.DoWithScope(ts, func() {
		r.providers = r.config.CreateProviders(ic, r.config.Hierarchy())
		r.defaultProviders = r.config.CreateProviders(ic, r.config.DefaultHierarchy())
	})
	r.variablesUsed = ts.GetRead()
}
