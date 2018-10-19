package lookup

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/functions"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
	"path/filepath"
)

type LookupKind string

const DATA_DIG = LookupKind(`data_dig`)
const DATA_HASH = LookupKind(`data_hash`)
const LOOKUP_KEY = LookupKind(`lookup_key`)

var FUNCTION_KEYS = []string{string(DATA_DIG), string(DATA_HASH), string(LOOKUP_KEY)}

type LocationKind string

const LC_PATH = LocationKind(`path`)
const LC_URI = LocationKind(`uri`)
const LC_GLOB = LocationKind(`glob`)
const LC_MAPPED_PATHS = LocationKind(`mapped_paths`)

var LOCATION_KEYS = []string{string(LC_PATH), `paths`, string(LC_GLOB), `globs`, string(LC_URI), `uris`, string(LC_MAPPED_PATHS)}

var RESERVED_OPTION_KEYS = []string{string(LC_PATH), string(LC_URI)}

type Function interface {
	Kind() LookupKind
	Name() string
}

type HierarchyEntry interface {
	Name() string
	Options() eval.KeyedValue
	DataDir() string
	Function() Function
}

type LookupInvocation interface {

}

type DataProvider interface {

}

type Config interface {
	Root() string
	Path() string
	LoadedConfig() eval.KeyedValue
	Defaults() HierarchyEntry
	Hierarchy() []HierarchyEntry
	DefaultHierarchy() []HierarchyEntry

	Resolve(c eval.Context) ResolvedConfig
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
	ReResolve(c eval.Context) ResolvedConfig
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

func (f *function) resolve(c eval.Context) *function {
	return f
}

type location struct {
	kind LookupKind
	name string
}

type Location interface {
	Kind() LocationKind
}

type path struct {
	path string
}

func (p* path) Kind() LocationKind {
	return LC_PATH
}

type glob struct {
	uri string
}

func (g* glob) Kind() LocationKind {
	return LC_GLOB
}

type uri struct {
	uri string
}

func (u* uri) Kind() LocationKind {
	return LC_URI
}

type mappedPaths struct {
	// Name of variable that contains an array of strings
	sourceVar string

	// Variable name to use when resolving template
	key string

	// Template containing interpolation of the key
	template string
}

func (m* mappedPaths) Kind() LocationKind {
	return LC_MAPPED_PATHS
}

type entry struct {
	name      string
	dataDir   string
	locations []Location
	options   *types.HashValue
	function  *function
}

func (e *entry) Name() string {
	return e.name
}

func (e *entry) Options() eval.KeyedValue {
	return e.options
}

func (e *entry) DataDir() string {
	return e.dataDir
}

func (e *entry) Function() Function {
	return e.function
}

type provider struct {

}

func (e *entry) createProvider(c eval.Context, defaults HierarchyEntry) DataProvider {
	// TODO: Create data provider base on entry
	return nil
}

func (e *entry) resolve(c eval.Context) *entry {
	// TODO: Resolve hiearchy entry with all interpolated strings etc.
	ce := &entry{}
	if e.function != nil {
		ce.function = e.function.resolve(c)
	}
	return ce
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
		hierarchy: []HierarchyEntry{&entry{name: `Common`, locations: []Location{&path{`common.yaml`}}}},
		defaultHierarchy: []HierarchyEntry{},
	}
}

type config struct {
	root          string
	path          string
	loadedHash    *types.HashValue
	defaults      HierarchyEntry
	hierarchy     []HierarchyEntry
	defaultHierarchy []HierarchyEntry
}

func NewConfig(c eval.Context, configPath string) Config {

	// TODO: Cache parsed file content
	if b, ok := types.BinaryFromFile2(c, configPath); ok {
		v, ok := eval.Load(c, eval.NewTypedName(eval.TYPE, `Hiera::Config`))
		if !ok {
			panic(eval.Error(eval.EVAL_FAILURE, issue.H{`message`: `Unable to load Hiera::Config data type`}))
		}
		cfgType := v.(eval.PType)
		yv := functions.UnmarshalYaml(c, b.Bytes())
		return createConfig(configPath, eval.AssertInstance(func() string {
				return fmt.Sprintf(`The Lookup Configuration at '%s'`, configPath)
			}, cfgType, yv).(*types.HashValue))
	}
	return DEFAULT_CONFIG
}

func (hc *config) Resolve(c eval.Context) ResolvedConfig {
	r := &resolvedConfig{config: hc}
	return r.ReResolve(c)
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

func (hc *config) LoadedConfig() eval.KeyedValue {
	return hc.loadedHash
}

func (hc *config) Defaults() HierarchyEntry {
	return hc.defaults
}

func (hc *config) createProviders(c eval.Context, hierarchy []HierarchyEntry) []DataProvider {
	providers := make([]DataProvider, len(hierarchy))
	defaults := hc.defaults.(*entry).resolve(c)
	for i, he := range hierarchy {
		providers[i] = he.(*entry).resolve(c).createProvider(c, defaults)
	}
	return providers
}

func createConfig(path string, hash *types.HashValue) Config {
	cfg := &config{root: filepath.Dir(path), path: path}

	var dflts HierarchyEntry
	if dv, ok := hash.Get4(`defaults`); ok {
		cfg.defaults = createHierarchyEntry(`defaults`, dv.(*types.HashValue), nil)
	} else {
		cfg.defaults = DEFAULT_CONFIG.Defaults()
	}

	if hv, ok := hash.Get4(`hierarchy`); ok {
		cfg.hierarchy = createHierarchy(hv.(*types.ArrayValue), dflts)
	} else {
		cfg.hierarchy = DEFAULT_CONFIG.Hierarchy()
	}

	if hv, ok := hash.Get4(`default_hierarchy`); ok {
		cfg.defaultHierarchy = createHierarchy(hv.(*types.ArrayValue), dflts)
	}

	return cfg
}

func createHierarchy(hier *types.ArrayValue, defaults HierarchyEntry) []HierarchyEntry {
	entries := make([]HierarchyEntry, 0, hier.Len())
	uniqueNames := make(map[string]bool, hier.Len())
	hier.Each(func( hv eval.PValue) {
		hh := hv.(*types.HashValue)
		name := hh.Get5(`name`, eval.EMPTY_STRING).String()
		if uniqueNames[name] {
			panic(eval.Error(HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED, issue.H{`name`: name}))
		}
		uniqueNames[name] = true
		entries = append(entries, createHierarchyEntry(name, hh, defaults))
	})
	return entries
}

func createHierarchyEntry(name string, hierEntry *types.HashValue, defaults HierarchyEntry) HierarchyEntry {
	entry := &entry{name: name}
	hierEntry.EachPair(func(k, v eval.PValue) {
		ks := k.String()
		if ks == `options` {
			entry.options = v.(*types.HashValue)
			entry.options.EachKey(func(optKey eval.PValue) {
				if utils.ContainsString(RESERVED_OPTION_KEYS, optKey.String()) {
					panic(eval.Error(HIERA_OPTION_RESERVED_BY_PUPPET, issue.H{`key`: optKey.String(), `name`: name}))
				}
			})
		} else if utils.ContainsString(FUNCTION_KEYS, ks) {
			if entry.function != nil {
				panic(eval.Error(HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS, issue.H{`keys`: FUNCTION_KEYS, `name`: name}))
			}
			entry.function = &function{LookupKind(ks), v.String()}
		} else if utils.ContainsString(LOCATION_KEYS, ks) {
			if entry.locations != nil {
				panic(eval.Error(HIERA_MULTIPLE_LOCATION_SPECS, issue.H{`keys`: LOCATION_KEYS, `name`: name}))
			}
			switch ks {
			case `path`:
				entry.locations = []Location{&path{v.String()}}
			case `paths`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]Location, 0, a.Len())
				a.Each(func(p eval.PValue) { entry.locations = append(entry.locations, &path{p.String()}) })
			case `glob`:
				entry.locations = []Location{&glob{v.String()}}
			case `globs`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]Location, 0, a.Len())
				a.Each(func(p eval.PValue) { entry.locations = append(entry.locations, &glob{p.String()}) })
			case `uri`:
				entry.locations = []Location{&uri{v.String()}}
			case `uris`:
				a := v.(*types.ArrayValue)
				entry.locations = make([]Location, 0, a.Len())
				a.Each(func(p eval.PValue) { entry.locations = append(entry.locations, &uri{p.String()}) })
			default: // Mapped paths
				a := v.(*types.ArrayValue)
				entry.locations = []Location{&mappedPaths{a.At(0).String(), a.At(1).String(), a.At(2).String()}}
			}
		}
	})

	if defaults != nil && entry.function == nil && defaults.Function() == nil {
		panic(eval.Error(HIERA_MISSING_DATA_PROVIDER_FUNCTION, issue.H{`keys`: FUNCTION_KEYS, `name`: name}))
	}
	return entry
}

type resolvedConfig struct {
	config *config
	variablesUsed map[string]eval.PValue
	providers []DataProvider
	defaultProviders []DataProvider
}

func (r *resolvedConfig) ReResolve(c eval.Context) ResolvedConfig {
	if r.variablesUsed == nil {
		r.resolve(c)
		return r
	}

	allEqual := true
	scope := c.Scope()
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
	rr.resolve(c)
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

func (r *resolvedConfig) resolve(c eval.Context) {
	ts := NewTrackingScope(c.Scope())
	c.DoWithScope(ts, func() {
		r.providers = r.config.createProviders(c, r.config.Hierarchy())
		r.defaultProviders = r.config.createProviders(c, r.config.DefaultHierarchy())
	})
	r.variablesUsed = ts.GetRead()
}