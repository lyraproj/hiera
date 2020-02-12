package internal

import (
	"fmt"
	"sync"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/hierasdk/hiera"
)

type dataHashProvider struct {
	hierarchyEntry api.Entry
	providerFunc   hiera.DataHash
	hashes         dgo.Map
	hashesLock     sync.RWMutex
}

func (dh *dataHashProvider) Hierarchy() api.Entry {
	return dh.hierarchyEntry
}

func (dh *dataHashProvider) LookupKey(key api.Key, ic api.Invocation, location api.Location) dgo.Value {
	root := key.Root()
	if value := dh.dataValue(ic, location, root); value != nil {
		ic.ReportFound(root, value)
		return value
	}
	ic.ReportNotFound(root)
	return nil
}

func (dh *dataHashProvider) dataValue(ic api.Invocation, location api.Location, root string) dgo.Value {
	value := dh.dataHash(ic, location).Get(root)
	if value == nil {
		return nil
	}
	return ic.Interpolate(value, true)
}

func (dh *dataHashProvider) providerFunction(ic api.Invocation) (pf hiera.DataHash) {
	if dh.providerFunc == nil {
		dh.providerFunc = dh.loadFunction(ic)
	}
	return dh.providerFunc
}

func (dh *dataHashProvider) loadFunction(ic api.Invocation) hiera.DataHash {
	n := dh.hierarchyEntry.Function().Name()
	switch n {
	case `yaml_data`:
		return provider.YamlData
	case `json_data`:
		return provider.JSONData
	}

	if fn, ok := ic.LoadFunction(dh.hierarchyEntry); ok {
		return func(pc hiera.ProviderContext) (value dgo.Map) {
			value = vf.Map()
			v := fn.Call(vf.MutableValues(pc))
			if dv, ok := v[0].(dgo.Map); ok {
				value = dv
			}
			return
		}
	}

	ic.ReportText(func() string { return fmt.Sprintf(`unresolved function '%s'`, n) })
	return func(pc hiera.ProviderContext) dgo.Map {
		return vf.Map()
	}
}

func (dh *dataHashProvider) dataHash(ic api.Invocation, location api.Location) (hash dgo.Map) {
	key := ``
	opts := dh.hierarchyEntry.Options()
	if location != nil {
		key = location.Resolved()
		opts = optionsWithLocation(opts, key)
	}

	var ok bool
	dh.hashesLock.RLock()
	hash, ok = dh.hashes.Get(key).(dgo.Map)
	dh.hashesLock.RUnlock()
	if ok {
		return
	}

	dh.hashesLock.Lock()
	defer dh.hashesLock.Unlock()

	if hash, ok = dh.hashes.Get(key).(dgo.Map); ok {
		return hash
	}
	hash = dh.providerFunction(ic)(ic.ServerContext(opts))
	dh.hashes.Put(key, hash)
	return
}

func (dh *dataHashProvider) FullName() string {
	return fmt.Sprintf(`data_hash function '%s'`, dh.hierarchyEntry.Function().Name())
}

// NewDataHashProvider creates a new provider with a data_hash function configured from the given entry
func NewDataHashProvider(he api.Entry) api.DataProvider {
	ls := he.Locations()
	return &dataHashProvider{hierarchyEntry: he, hashes: vf.MapWithCapacity(len(ls))}
}

func optionsWithLocation(options dgo.Map, loc string) dgo.Map {
	return options.Merge(vf.Map(`path`, loc))
}
