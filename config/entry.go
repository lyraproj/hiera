package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lyraproj/dgo/vf"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/hiera/hieraapi"
)

type (
	entry struct {
		cfg        *hieraCfg
		dataDir    string
		pluginDir  string
		pluginFile string
		options    dgo.Map
		function   hieraapi.Function
		name       string
		locations  []hieraapi.Location
	}
)

// FunctionKeys are the valid keys to use when defining a function in a hierarchy entry
var FunctionKeys = []string{string(hieraapi.KindDataDig), string(hieraapi.KindDataHash), string(hieraapi.KindLookupKey)}

// LocationKeys are the valid keys to use when defining locations in a hierarchy entry
var LocationKeys = []string{
	string(hieraapi.LcPath), `paths`,
	string(hieraapi.LcGlob), `globs`,
	string(hieraapi.LcURI), `uris`,
	string(hieraapi.LcMappedPaths)}

// ReservedOptionKeys are the option keys that are reserved by Hiera
var ReservedOptionKeys = []string{string(hieraapi.LcPath), string(hieraapi.LcURI)}

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

func (e *entry) Function() hieraapi.Function {
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
			e.function = &function{hieraapi.FunctionKind(ks), v.String()}
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

func (e *entry) resolveFunction(ic hieraapi.Invocation, defaults hieraapi.Entry) {
	if e.function == nil {
		if defaults == nil {
			e.function = &function{kind: hieraapi.KindDataHash, name: `yaml_data`}
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

func (e *entry) resolveDataDir(ic hieraapi.Invocation, defaults hieraapi.Entry) {
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

func (e *entry) resolvePluginDir(ic hieraapi.Invocation, defaults hieraapi.Entry) {
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

func (e *entry) resolveOptions(ic hieraapi.Invocation, defaults hieraapi.Entry) {
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

func (e *entry) resolveLocations(ic hieraapi.Invocation) {
	var dataRoot string
	if filepath.IsAbs(e.dataDir) {
		dataRoot = e.dataDir
	} else {
		dataRoot = filepath.Join(e.cfg.root, e.dataDir)
	}
	if e.locations != nil {
		ne := make([]hieraapi.Location, 0, len(e.locations))
		for _, l := range e.locations {
			ne = append(ne, l.Resolve(ic, dataRoot)...)
		}
		e.locations = ne
	}
}

func (e *entry) Resolve(ic hieraapi.Invocation, defaults hieraapi.Entry) hieraapi.Entry {
	// Resolve interpolated strings and locations
	ce := *e

	ce.resolveFunction(ic, defaults)
	ce.resolveDataDir(ic, defaults)
	ce.resolvePluginDir(ic, defaults)
	ce.resolveOptions(ic, defaults)
	ce.resolveLocations(ic)

	return &ce
}
