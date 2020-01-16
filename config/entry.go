package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lyraproj/dgo/vf"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/hiera/api"
)

type (
	entry struct {
		cfg        *hieraCfg
		dataDir    string
		pluginDir  string
		pluginFile string
		options    dgo.Map
		function   api.Function
		name       string
		locations  []api.Location
	}
)

// FunctionKeys are the valid keys to use when defining a function in a hierarchy entry
var FunctionKeys = []string{string(api.KindDataDig), string(api.KindDataHash), string(api.KindLookupKey)}

// LocationKeys are the valid keys to use when defining locations in a hierarchy entry
var LocationKeys = []string{
	string(api.LcPath), `paths`,
	string(api.LcGlob), `globs`,
	string(api.LcURI), `uris`,
	string(api.LcMappedPaths)}

// ReservedOptionKeys are the option keys that are reserved by Hiera
var ReservedOptionKeys = []string{string(api.LcPath), string(api.LcURI)}

func (e *entry) Options() dgo.Map {
	return e.options
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

func (e *entry) Function() api.Function {
	return e.function
}

func (e *entry) initialize(name string, entryHash dgo.Map) {
	entryHash.EachEntry(func(me dgo.MapEntry) {
		ks := me.Key().String()
		v := me.Value()
		if ks == `options` {
			e.options = v.(dgo.Map)
			e.options.EachKey(func(optKey dgo.Value) {
				if util.ContainsString(ReservedOptionKeys, optKey.String()) {
					panic(
						fmt.Errorf(`option key '%s' used in hierarchy '%s' is reserved by Hiera`, optKey.String(), name))
				}
			})
		} else if util.ContainsString(FunctionKeys, ks) {
			if e.function != nil {
				panic(fmt.Errorf(`only one of %s can be defined in hierarchy '%s'`, strings.Join(FunctionKeys, `, `), name))
			}
			e.function = &function{api.FunctionKind(ks), v.String()}
		}
	})
}

func (e *entry) Copy(cfg api.Config) api.Entry {
	c := *e
	c.cfg = cfg.(*hieraCfg)
	return &c
}

func (e *entry) Name() string {
	return e.name
}

func (e *entry) Locations() []api.Location {
	return e.locations
}

func (e *entry) resolveFunction(ic api.Invocation, defaults api.Entry) {
	if e.function == nil {
		if defaults == nil {
			e.function = &function{kind: api.KindDataHash, name: `yaml_data`}
		} else {
			e.function = defaults.Function()
		}
	} else if f, fc := e.function.Resolve(ic); fc {
		e.function = f
	}

	if e.function == nil {
		panic(fmt.Errorf(`one of %s must be defined in hierarchy '%s'`, strings.Join(FunctionKeys, `, `), e.name))
	}
}

func (e *entry) resolveDataDir(ic api.Invocation, defaults api.Entry) {
	e.resolveFunction(ic, defaults)
	if e.dataDir == `` {
		if defaults == nil {
			e.dataDir = defaultDataDir()
		} else {
			e.dataDir = defaults.DataDir()
		}
	} else {
		if d, dc := ic.InterpolateString(e.dataDir, false); dc {
			e.dataDir = d.String()
		}
	}
}

func (e *entry) resolvePluginDir(ic api.Invocation, defaults api.Entry) {
	if e.pluginDir == `` {
		if defaults == nil {
			e.pluginDir = defaultPluginDir()
		} else {
			e.pluginDir = defaults.PluginDir()
		}
	} else {
		if d, dc := ic.InterpolateString(e.pluginDir, false); dc {
			e.pluginDir = d.String()
		}
	}
	if !filepath.IsAbs(e.pluginDir) {
		e.pluginDir = filepath.Join(e.cfg.root, e.pluginDir)
	}
}

func (e *entry) resolveOptions(ic api.Invocation, defaults api.Entry) {
	if e.options == nil {
		if defaults != nil {
			e.options = defaults.Options()
		}
	} else if e.options.Len() > 0 {
		e.options = ic.Interpolate(e.options, false).(dgo.Map)
	}
	if e.options == nil {
		e.options = vf.Map()
	}
}

func (e *entry) resolveLocations(ic api.Invocation) {
	var dataRoot string
	if filepath.IsAbs(e.dataDir) {
		dataRoot = e.dataDir
	} else {
		dataRoot = filepath.Join(e.cfg.root, e.dataDir)
	}
	if e.locations != nil {
		ne := make([]api.Location, 0, len(e.locations))
		for _, l := range e.locations {
			ne = append(ne, l.Resolve(ic, dataRoot)...)
		}
		e.locations = ne
	}
}

func (e *entry) Resolve(ic api.Invocation, defaults api.Entry) api.Entry {
	// Resolve interpolated strings and locations
	ce := *e

	ce.resolveFunction(ic, defaults)
	ce.resolveDataDir(ic, defaults)
	ce.resolvePluginDir(ic, defaults)
	ce.resolveOptions(ic, defaults)
	ce.resolveLocations(ic)

	return &ce
}
