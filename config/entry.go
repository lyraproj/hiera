package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/dgo/vf"
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
var LocationKeys = []string{string(hieraapi.LcPath), `paths`, string(hieraapi.LcGlob), `globs`, string(hieraapi.LcURI), `uris`, string(hieraapi.LcMappedPaths)}

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
		panic(fmt.Errorf(`one of %s must be defined in hierarchy '%s'`, strings.Join(FunctionKeys, `, `), e.name))
	}

	if ce.dataDir == `` {
		if defaults == nil {
			ce.dataDir = defaultDataDir()
		} else {
			ce.dataDir = defaults.DataDir()
		}
	} else {
		if d, dc := ic.InterpolateString(ce.dataDir, false); dc {
			ce.dataDir = d.String()
		}
	}

	if ce.pluginDir == `` {
		if defaults == nil {
			ce.pluginDir = defaultPluginDir()
		} else {
			ce.pluginDir = defaults.PluginDir()
		}
	} else {
		if d, dc := ic.InterpolateString(ce.pluginDir, false); dc {
			ce.pluginDir = d.String()
		}
	}
	if !filepath.IsAbs(ce.pluginDir) {
		ce.pluginDir = filepath.Join(e.cfg.root, ce.pluginDir)
	}

	if ce.options == nil {
		if defaults != nil {
			ce.options = defaults.Options()
		}
	} else if ce.options.Len() > 0 {
		ce.options = ic.Interpolate(ce.options, false).(dgo.Map)
	}
	if ce.options == nil {
		ce.options = vf.Map()
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
