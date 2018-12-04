package impl

import (
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-evaluator/utils"
	"github.com/lyraproj/hiera/config"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/issue/issue"
	"path/filepath"

	// Ensure that pcore is initialized
	_ "github.com/lyraproj/puppet-evaluator/pcore"
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
	dataDir   string
	options   eval.OrderedMap
	function  config.Function
}

func (e *entry) Options() eval.OrderedMap {
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
	case config.DATA_HASH:
		return newDataHashProvider(ic, e)
	case config.DATA_DIG:
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
		panic(eval.Error(HIERA_MISSING_DATA_PROVIDER_FUNCTION, issue.H{`keys`: config.FUNCTION_KEYS, `name`: e.name}))
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
			ce.options = o.(*types.HashValue)
		}
	}

	if e.locations != nil {
		ne := make([]lookup.Location, 0, len(e.locations))
		ce.locations = ne
		for _, l := range e.locations {
			ne = append(ne, l.Resolve(ic, ce.dataDir)...)
		}
	}

	return &ce
}

var hieraTypeSet eval.TypeSet

var DEFAULT_CONFIG config.Config

func init() {
	hieraTypeSet = eval.NewTypeSet(`Hiera`, `{
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

	DEFAULT_CONFIG = &hieraCfg{
		root: ``,
		path: ``,
		loadedHash: nil,
		defaults: &entry{dataDir: `data`, function: &function{kind: config.DATA_HASH, name: `yaml_data`}},
		hierarchy: []config.HierarchyEntry{&hierEntry{name: `Common`, locations: []lookup.Location{&path{original: `common.yaml`}}}},
		defaultHierarchy: []config.HierarchyEntry{},
	}

	eval.Puppet.DefineSetting(`hiera_config`, types.DefaultStringType(), nil)
}

type hieraCfg struct {
	root          string
	path          string
	loadedHash    eval.OrderedMap
	defaults      config.Entry
	hierarchy     []config.HierarchyEntry
	defaultHierarchy []config.HierarchyEntry
}

func NewConfig(ic lookup.Invocation, configPath string) config.Config {

	// TODO: Cache parsed file content
	if b, ok := types.BinaryFromFile2(ic, configPath); ok {
		v, ok := eval.Load(ic, eval.NewTypedName(eval.NsType, `Hiera::Config`))
		if !ok {
			panic(eval.Error(eval.EVAL_FAILURE, issue.H{`message`: `Unable to load Hiera::Config data type`}))
		}
		cfgType := v.(eval.Type)
		yv := UnmarshalYaml(ic, b.Bytes())
		return createConfig(ic, configPath, eval.AssertInstance(func() string {
				return fmt.Sprintf(`The Lookup Configuration at '%s'`, configPath)
			}, cfgType, yv).(*types.HashValue))
	}
	return DEFAULT_CONFIG
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

func (hc *hieraCfg) LoadedConfig() eval.OrderedMap {
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

func createConfig(ic lookup.Invocation, path string, hash *types.HashValue) config.Config {
	cfg := &hieraCfg{root: filepath.Dir(path), path: path}

	if dv, ok := hash.Get4(`defaults`); ok {
		cfg.defaults = createDefaultsEntry(ic, dv.(*types.HashValue))
	} else {
		cfg.defaults = DEFAULT_CONFIG.Defaults()
	}

	if hv, ok := hash.Get4(`hierarchy`); ok {
		cfg.hierarchy = createHierarchy(ic, hv.(*types.ArrayValue))
	} else {
		cfg.hierarchy = DEFAULT_CONFIG.Hierarchy()
	}

	if hv, ok := hash.Get4(`default_hierarchy`); ok {
		cfg.defaultHierarchy = createHierarchy(ic, hv.(*types.ArrayValue))
	}

	return cfg
}

func createHierarchy(ic lookup.Invocation, hier *types.ArrayValue) []config.HierarchyEntry {
	entries := make([]config.HierarchyEntry, 0, hier.Len())
	uniqueNames := make(map[string]bool, hier.Len())
	hier.Each(func( hv eval.Value) {
		hh := hv.(*types.HashValue)
		name := hh.Get5(`name`, eval.EMPTY_STRING).String()
		if uniqueNames[name] {
			panic(eval.Error(HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED, issue.H{`name`: name}))
		}
		uniqueNames[name] = true
		entries = append(entries, createHierarchyEntry(ic, name, hh))
	})
	return entries
}

func (entry* entry) initialize(ic lookup.Invocation, name string, entryHash *types.HashValue) {
	entryHash.EachPair(func(k, v eval.Value) {
		ks := k.String()
		if ks == `options` {
			entry.options = v.(*types.HashValue)
			entry.options.EachKey(func(optKey eval.Value) {
				if utils.ContainsString(config.RESERVED_OPTION_KEYS, optKey.String()) {
					panic(eval.Error(HIERA_OPTION_RESERVED_BY_PUPPET, issue.H{`key`: optKey.String(), `name`: name}))
				}
			})
		} else if utils.ContainsString(config.FUNCTION_KEYS, ks) {
			if entry.function != nil {
				panic(eval.Error(HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS, issue.H{`keys`: config.FUNCTION_KEYS, `name`: name}))
			}
			entry.function = &function{config.LookupKind(ks), v.String()}
		}
	})
}

func createDefaultsEntry(ic lookup.Invocation, entryHash *types.HashValue) config.Entry {
	defaults := &entry{}
	defaults.initialize(ic, `defaults`, entryHash)
	return defaults
}

func createHierarchyEntry(ic lookup.Invocation, name string, entryHash *types.HashValue) config.HierarchyEntry {
	entry := &hierEntry{name: name}
	entry.initialize(ic, name, entryHash)
	entryHash.EachPair(func(k, v eval.Value) {
		ks := k.String()
		if utils.ContainsString(config.LOCATION_KEYS, ks) {
			if entry.locations != nil {
				panic(eval.Error(HIERA_MULTIPLE_LOCATION_SPECS, issue.H{`keys`: config.LOCATION_KEYS, `name`: name}))
			}
			switch ks {
			case `path`:
				entry.locations = []lookup.Location{&path{original: v.String()}}
			case `paths`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]lookup.Location, 0, a.Len())
				a.Each(func(p eval.Value) { entry.locations = append(entry.locations, &path{original:p.String()}) })
			case `glob`:
				entry.locations = []lookup.Location{&glob{v.String()}}
			case `globs`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]lookup.Location, 0, a.Len())
				a.Each(func(p eval.Value) { entry.locations = append(entry.locations, &glob{p.String()}) })
			case `uri`:
				entry.locations = []lookup.Location{&uri{original: v.String()}}
			case `uris`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]lookup.Location, 0, a.Len())
				a.Each(func(p eval.Value) { entry.locations = append(entry.locations, &uri{original:p.String()}) })
			default: // Mapped paths
				a := v.(*types.ArrayValue)
				entry.locations = []lookup.Location{&mappedPaths{a.At(0).String(), a.At(1).String(), a.At(2).String()}}
			}
		}
	})
	return entry
}

type resolvedConfig struct {
	config *hieraCfg
	variablesUsed map[string]eval.Value
	providers []lookup.DataProvider
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
		if sv, ok := scope.Get(k); ok && v.Equals(sv, nil) {
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