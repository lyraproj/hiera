package provider

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lyraproj/hiera/config"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hierasdk/hiera"
)

// ModulePath is the session option that the ModuleLookupKey function uses as the module path. The
// value must be a string with paths separated with the OS-specific path separator.
const ModulePath = `hiera::lookup::modulepath`

// ModuleLookupKey is a lookup key function that performs a lookup in modules. The function expects
// a key with multiple segments separated by a double colon '::'. The first segment is considered
// the name of a module and that module must be found in the path stored as the ModulePath option.
// If such a path exists and is a directory that in turn contains a hiera.yaml file, then a lookup
// will be performed in that module.
func ModuleLookupKey(pc hiera.ProviderContext, key string) dgo.Value {
	sc := pc.(api.ServerContext)
	if ci := strings.Index(key, `::`); ci > 0 {
		modName := strings.ToLower(key[:ci])
		var mp dgo.Function
		mpm := moduleProviders(sc)
		if f := mpm.Get(modName); f != nil {
			mp = f.(dgo.Function)
		} else {
			mp = loadModuleProvider(sc.Invocation(), mpm, modName)
		}
		iv := sc.Invocation()
		return iv.WithModule(modName, func() dgo.Value {
			if mp == notFoundLookupKeyFunc {
				iv.ReportModuleNotFound()
				return nil
			}
			return mp.Call(vf.MutableValues(pc, key))[0]
		})
	}
	return nil
}

func moduleProviders(sc api.ServerContext) dgo.Map {
	var mp dgo.Map
	if c, ok := sc.CachedValue(`hiera::moduleproviders`); ok {
		mp = c.(dgo.Map)
	} else {
		mp = vf.MutableMap()
		sc.Cache(`hiera::moduleproviders`, mp)
	}
	return mp
}

var notFoundLookupKeyFunc = vf.Value(func(pc hiera.ProviderContext, key string) dgo.Value { return nil }).(dgo.Function)

func loadModuleProvider(ic api.Invocation, mpm dgo.Map, moduleName string) dgo.Function {
	var mp dgo.Function = notFoundLookupKeyFunc
	if modulePath, ok := ic.SessionOptions().Get(ModulePath).(dgo.String); ok {
		for _, path := range filepath.SplitList(modulePath.GoString()) {
			if loaded := loadModule(path, moduleName); loaded != nil {
				mp = loaded
				break
			}
		}
	}
	mpm.Put(moduleName, mp)
	return mp
}

func loadModule(path, moduleName string) dgo.Function {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		panic(err)
	}

	// Lookup module by finding a directory that matches the moduleName using case insensitive string comparison
	fileInfos, err := f.Readdir(-1)
	_ = f.Close()
	if err != nil {
		panic(err)
	}

	for _, fi := range fileInfos {
		if !strings.EqualFold(fi.Name(), moduleName) {
			continue
		}
		if !fi.IsDir() {
			break
		}
		configPath := filepath.Join(path, fi.Name(), config.FileName)
		cf, err := os.Lstat(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			panic(err)
		}
		if cf.IsDir() {
			break
		}
		return loadHieraConfig(configPath, moduleName)
	}
	return nil
}

func loadHieraConfig(configPath, moduleName string) dgo.Function {
	return vf.Value(
		func(pc hiera.ProviderContext, key string) dgo.Value {
			if sc, ok := pc.(api.ServerContext); ok {
				return ConfigLookupKeyAt(sc, configPath, key, moduleName)
			}
			return nil
		}).(dgo.Function)
}
