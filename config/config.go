// Package config contains the code to load and resolve the Hiera configuration
package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/tf"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/dgoyaml/yaml"
	"github.com/lyraproj/hiera/api"
)

type (
	hieraCfg struct {
		root             string
		path             string
		defaults         *entry
		hierarchy        []api.Entry
		defaultHierarchy []api.Entry
	}
)

const definitions = `{
	options=map[/\A[A-Za-z](:?[0-9A-Za-z_-]*[0-9A-Za-z])?\z/]data,
	rstring=string[1],
	defaults={
	  options?:options,
	  data_dig?:rstring,
	  data_hash?:rstring,
	  lookup_key?:rstring,
	  datadir?:rstring,
	  plugindir?:rstring
	},
	entry={
	  name:rstring,
	  options?:options,
	  data_dig?:rstring,
	  data_hash?:rstring,
	  lookup_key?:rstring,
	  datadir?:rstring,
	  plugindir?:rstring,
	  pluginfile?:rstring,
	  path?:rstring,
	  paths?:[1]rstring,
	  glob?:rstring,
	  globs?:[1]rstring,
	  uri?:rstring,
	  uris?:[1]rstring,
	  mapped_paths?:[3,3]rstring
	}
}`

const hieraTypeString = `{
	version:5,
	defaults?:defaults,
	hierarchy?:[]entry,
	default_hierarchy?:[]entry
}`

// FileName is the default file name for the Hiera configuration file.
const FileName = `hiera.yaml`

var cfgType dgo.Type

func init() {
	tf.ParseType(definitions)
	cfgType = tf.ParseType(hieraTypeString)
}

// New creates a new unresolved Config from the given path. If the path does not exist, the
// default config is returned.
func New(configPath string) api.Config {
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		dc := &hieraCfg{
			root:             filepath.Dir(configPath),
			path:             ``,
			defaultHierarchy: []api.Entry{},
		}
		dc.defaults = dc.makeDefaultConfig()
		dc.hierarchy = dc.makeDefaultHierarchy()
		return dc
	}

	yv, err := yaml.Unmarshal(content)
	if err != nil {
		panic(err)
	}
	cfgMap := yv.(dgo.Map)
	if !cfgType.Instance(cfgMap) {
		panic(tf.IllegalAssignment(cfgType, cfgMap))
	}

	return createConfig(configPath, cfgMap)
}

func createConfig(path string, hash dgo.Map) api.Config {
	cfg := &hieraCfg{root: filepath.Dir(path), path: path}

	if dv := hash.Get(`defaults`); dv != nil {
		cfg.defaults = cfg.createEntry(`defaults`, dv.(dgo.Map)).(*entry)
	} else {
		cfg.defaults = cfg.makeDefaultConfig()
	}

	if hv := hash.Get(`hierarchy`); hv != nil {
		cfg.hierarchy = cfg.createHierarchy(hv.(dgo.Array))
	} else {
		cfg.hierarchy = cfg.makeDefaultHierarchy()
	}

	if hv := hash.Get(`default_hierarchy`); hv != nil {
		cfg.defaultHierarchy = cfg.createHierarchy(hv.(dgo.Array))
	}

	return cfg
}

func defaultDataDir() string {
	dataDir, exists := os.LookupEnv("HIERA_DATADIR")
	if !exists {
		dataDir = `data`
	}
	return dataDir
}

func defaultPluginDir() string {
	pluginDir, exists := os.LookupEnv("HIERA_PLUGINDIR")
	if !exists {
		pluginDir = `plugin`
	}
	return pluginDir
}

func (hc *hieraCfg) makeDefaultConfig() *entry {
	return &entry{
		cfg:       hc,
		dataDir:   defaultDataDir(),
		pluginDir: defaultPluginDir(),
		function:  &function{kind: api.KindDataHash, name: `yaml_data`},
	}
}

func (hc *hieraCfg) makeDefaultHierarchy() []api.Entry {
	return []api.Entry{
		// The lyra default behavior is to look for a <Hiera root>/data.yaml. Hiera root is the current directory.
		&entry{cfg: hc, dataDir: `.`, name: `Root`, locations: []api.Location{NewPath(`data.yaml`)}},
		// Hiera proper default behavior is to look for <Hiera root>/data/common.yaml
		&entry{cfg: hc, name: `Common`, locations: []api.Location{NewPath(`common.yaml`)}}}
}

func (hc *hieraCfg) Hierarchy() []api.Entry {
	return hc.hierarchy
}

func (hc *hieraCfg) DefaultHierarchy() []api.Entry {
	return hc.defaultHierarchy
}

func (hc *hieraCfg) Root() string {
	return hc.root
}

func (hc *hieraCfg) Path() string {
	return hc.path
}

func (hc *hieraCfg) Defaults() api.Entry {
	return hc.defaults
}

func (hc *hieraCfg) createHierarchy(hierarchy dgo.Array) []api.Entry {
	entries := make([]api.Entry, 0, hierarchy.Len())
	uniqueNames := make(map[string]bool, hierarchy.Len())
	hierarchy.Each(func(hv dgo.Value) {
		hh := hv.(dgo.Map)
		name := ``
		if nv := hh.Get(`name`); nv != nil {
			name = nv.String()
		}
		if uniqueNames[name] {
			panic(fmt.Errorf(`hierarchy name '%s' defined more than once`, name))
		}
		uniqueNames[name] = true
		entries = append(entries, hc.createEntry(name, hh))
	})
	return entries
}

func (hc *hieraCfg) createEntry(name string, entryHash dgo.Map) api.Entry {
	entry := &entry{cfg: hc, name: name}
	entry.initialize(name, entryHash)
	entryHash.EachEntry(func(me dgo.MapEntry) {
		ks := me.Key().String()
		v := me.Value()
		switch {
		case ks == `datadir`:
			entry.dataDir = v.String()
		case ks == `plugindir`:
			entry.pluginDir = v.String()
		case ks == `pluginfile`:
			entry.pluginFile = v.String()
		case util.ContainsString(LocationKeys, ks):
			if entry.locations != nil {
				panic(fmt.Errorf(`only one of %s can be defined in hierarchy '%s'`, strings.Join(LocationKeys, `, `), name))
			}
			switch ks {
			case `path`:
				entry.locations = []api.Location{NewPath(v.String())}
			case `paths`:
				a := v.(dgo.Array)
				entry.locations = make([]api.Location, 0, a.Len())
				a.Each(func(p dgo.Value) { entry.locations = append(entry.locations, NewPath(p.String())) })
			case `glob`:
				entry.locations = []api.Location{NewGlob(v.String())}
			case `globs`:
				a := v.(dgo.Array)
				entry.locations = make([]api.Location, 0, a.Len())
				a.Each(func(p dgo.Value) { entry.locations = append(entry.locations, NewGlob(p.String())) })
			case `uri`:
				entry.locations = []api.Location{NewURI(v.String())}
			case `uris`:
				a := v.(dgo.Array)
				entry.locations = make([]api.Location, 0, a.Len())
				a.Each(func(p dgo.Value) { entry.locations = append(entry.locations, NewURI(p.String())) })
			default: // Mapped paths
				a := v.(dgo.Array)
				entry.locations = []api.Location{NewMappedPaths(a.Get(0).String(), a.Get(1).String(), a.Get(2).String())}
			}
		}
	})
	return entry
}
