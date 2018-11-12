package lookup

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
	"path/filepath"

	// Ensure that pcore is initialized
	_ "github.com/puppetlabs/go-evaluator/pcore"
)

type LookupKind string

const DATA_DIG = LookupKind(`data_dig`)
const DATA_HASH = LookupKind(`data_hash`)
const LOOKUP_KEY = LookupKind(`lookup_key`)

var FUNCTION_KEYS = []string{string(DATA_DIG), string(DATA_HASH), string(LOOKUP_KEY)}

var LOCATION_KEYS = []string{string(LC_PATH), `paths`, string(LC_GLOB), `globs`, string(LC_URI), `uris`, string(LC_MAPPED_PATHS)}

var RESERVED_OPTION_KEYS = []string{string(LC_PATH), string(LC_URI)}

type Function interface {
	Kind() LookupKind
	Name() string
	resolve(ic Invocation) (Function, bool)
}

type Entry interface {
	Options() eval.OrderedMap
	DataDir() string
	Function() Function
}

type HierarchyEntry interface {
	Entry
	Name() string
	resolve(ic Invocation, defaults Entry) HierarchyEntry
	createProvider(ic Invocation) DataProvider
}

type Config interface {
	Root() string
	Path() string
	LoadedConfig() eval.OrderedMap
	Defaults() Entry
	Hierarchy() []HierarchyEntry
	DefaultHierarchy() []HierarchyEntry

	Resolve(ic Invocation) ResolvedConfig
}

type ResolvedConfig interface {
	// Config returns the original Config that the receiver was created from
	Config() Config

	// Hierarchy returns the DataProvider slice
	Hierarchy() []DataProvider

	// DefaultHierarchy returns the DataProvider slice for the configured default_hierarchy.
	// The slice will be empty if no such hierarchy has been defined.
	DefaultHierarchy() []DataProvider

	// ReResolve resolves the already resolved receiver using the eval.Scope currently
	// held by the given eval.Context. The receiver will return itself when all variables
	// in the given scope still contains the exact same values as the scope used when the
	// receiver was created,
	ReResolve(ic Invocation) ResolvedConfig
}

type function struct {
	kind LookupKind
	name string
}

func (f *function) Kind() LookupKind {
	return f.kind
}

func (f *function) Name() string {
	return f.name
}

func (f *function) resolve(ic Invocation) (Function, bool) {
	if n, changed := interpolateString(ic, f.name, false); changed {
		return &function{f.kind, n.String()}, true
	}
	return f, false
}

type entry struct {
	dataDir   string
	options   eval.OrderedMap
	function  Function
}

func (e *entry) Options() eval.OrderedMap {
	return e.options
}

func (e *entry) DataDir() string {
	return e.dataDir
}

func (e *entry) Function() Function {
	return e.function
}

type hierEntry struct {
	entry
	name      string
	locations []Location
}

func (e *hierEntry) Name() string {
	return e.name
}

func (e *hierEntry) createProvider(ic Invocation) DataProvider {
	switch e.function.Kind() {
	case DATA_HASH:
		return newDataHashProvider(ic, e)
	case DATA_DIG:
		return newDataDigProvider(ic, e)
	default:
		return newLookupKeyProvider(ic, e)
	}
}

func (e *hierEntry) resolve(ic Invocation, defaults Entry) HierarchyEntry {
	// Resolve interpolated strings and locations
	ce := *e

	if e.function == nil {
		e.function = defaults.Function()
	} else if f, fc := e.function.resolve(ic); fc {
		ce.function = f
	}

	if e.function == nil {
		panic(eval.Error(HIERA_MISSING_DATA_PROVIDER_FUNCTION, issue.H{`keys`: FUNCTION_KEYS, `name`: e.name}))
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
		ne := make([]Location, 0, len(e.locations))
		ce.locations = ne
		for _, l := range e.locations {
			ne = append(ne, l.resolve(ic, ce.dataDir)...)
		}
	}

	return &ce
}

var hieraTypeSet eval.TypeSet

var DEFAULT_CONFIG Config

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

	DEFAULT_CONFIG = &config{
		root: ``,
		path: ``,
		loadedHash: nil,
		defaults: &entry{dataDir: `data`, function: &function{kind: DATA_HASH, name: `yaml_data`}},
		hierarchy: []HierarchyEntry{&hierEntry{name: `Common`, locations: []Location{&path{original:`common.yaml`}}}},
		defaultHierarchy: []HierarchyEntry{},
	}

	eval.Puppet.DefineSetting(`hiera_config`, types.DefaultStringType(), nil)
}

type config struct {
	root          string
	path          string
	loadedHash    eval.OrderedMap
	defaults      Entry
	hierarchy     []HierarchyEntry
	defaultHierarchy []HierarchyEntry
}

func NewConfig(ic Invocation, configPath string) Config {

	// TODO: Cache parsed file content
	if b, ok := types.BinaryFromFile2(ic, configPath); ok {
		v, ok := eval.Load(ic, eval.NewTypedName(eval.TYPE, `Hiera::Config`))
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

func (hc *config) Resolve(ic Invocation) ResolvedConfig {
	r := &resolvedConfig{config: hc}
	return r.ReResolve(ic)
}

func (hc *config) Hierarchy() []HierarchyEntry {
	return hc.hierarchy
}

func (hc *config) DefaultHierarchy() []HierarchyEntry {
	return hc.defaultHierarchy
}

func (hc *config) Root() string {
	return hc.root
}

func (hc *config) Path() string {
	return hc.path
}

func (hc *config) LoadedConfig() eval.OrderedMap {
	return hc.loadedHash
}

func (hc *config) Defaults() Entry {
	return hc.defaults
}

func (hc *config) createProviders(ic Invocation, hierarchy []HierarchyEntry) []DataProvider {
	providers := make([]DataProvider, len(hierarchy))
	defaults := hc.defaults.(*hierEntry).resolve(ic, nil)
	for i, he := range hierarchy {
		providers[i] = he.(*hierEntry).resolve(ic, defaults).createProvider(ic)
	}
	return providers
}

func createConfig(ic Invocation, path string, hash *types.HashValue) Config {
	cfg := &config{root: filepath.Dir(path), path: path}

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

func createHierarchy(ic Invocation, hier *types.ArrayValue) []HierarchyEntry {
	entries := make([]HierarchyEntry, 0, hier.Len())
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

func (entry* entry) initialize(ic Invocation, name string, entryHash *types.HashValue) {
	entryHash.EachPair(func(k, v eval.Value) {
		ks := k.String()
		if ks == `options` {
			entry.options = v.(*types.HashValue)
			entry.options.EachKey(func(optKey eval.Value) {
				if utils.ContainsString(RESERVED_OPTION_KEYS, optKey.String()) {
					panic(eval.Error(HIERA_OPTION_RESERVED_BY_PUPPET, issue.H{`key`: optKey.String(), `name`: name}))
				}
			})
		} else if utils.ContainsString(FUNCTION_KEYS, ks) {
			if entry.function != nil {
				panic(eval.Error(HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS, issue.H{`keys`: FUNCTION_KEYS, `name`: name}))
			}
			entry.function = &function{LookupKind(ks), v.String()}
		}
	})
}

func createDefaultsEntry(ic Invocation, entryHash *types.HashValue) Entry {
	defaults := &entry{}
	defaults.initialize(ic, `defaults`, entryHash)
	return defaults
}

func createHierarchyEntry(ic Invocation, name string, entryHash *types.HashValue) HierarchyEntry {
	entry := &hierEntry{name: name}
	entry.initialize(ic, name, entryHash)
	entryHash.EachPair(func(k, v eval.Value) {
		ks := k.String()
		if utils.ContainsString(LOCATION_KEYS, ks) {
			if entry.locations != nil {
				panic(eval.Error(HIERA_MULTIPLE_LOCATION_SPECS, issue.H{`keys`: LOCATION_KEYS, `name`: name}))
			}
			switch ks {
			case `path`:
				entry.locations = []Location{&path{original:v.String()}}
			case `paths`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]Location, 0, a.Len())
				a.Each(func(p eval.Value) { entry.locations = append(entry.locations, &path{original:p.String()}) })
			case `glob`:
				entry.locations = []Location{&glob{v.String()}}
			case `globs`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]Location, 0, a.Len())
				a.Each(func(p eval.Value) { entry.locations = append(entry.locations, &glob{p.String()}) })
			case `uri`:
				entry.locations = []Location{&uri{original:v.String()}}
			case `uris`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]Location, 0, a.Len())
				a.Each(func(p eval.Value) { entry.locations = append(entry.locations, &uri{original:p.String()}) })
			default: // Mapped paths
				a := v.(*types.ArrayValue)
				entry.locations = []Location{&mappedPaths{a.At(0).String(), a.At(1).String(), a.At(2).String()}}
			}
		}
	})
	return entry
}

type resolvedConfig struct {
	config *config
	variablesUsed map[string]eval.Value
	providers []DataProvider
	defaultProviders []DataProvider
}

func (r *resolvedConfig) ReResolve(ic Invocation) ResolvedConfig {
	if r.variablesUsed == nil {
		r.resolve(ic)
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
	rr.resolve(ic)
	return rr
}

func (r *resolvedConfig) Config() Config {
	return r.config
}

func (r *resolvedConfig) Hierarchy() []DataProvider {
	return r.providers
}

func (r *resolvedConfig) DefaultHierarchy() []DataProvider {
	return r.defaultProviders
}

func (r *resolvedConfig) resolve(ic Invocation) {
	ts := NewTrackingScope(ic.Scope())
	ic.DoWithScope(ts, func() {
		r.providers = r.config.createProviders(ic, r.config.Hierarchy())
		r.defaultProviders = r.config.createProviders(ic, r.config.DefaultHierarchy())
	})
	r.variablesUsed = ts.GetRead()
}