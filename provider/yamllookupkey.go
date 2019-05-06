package provider

import (
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
)

var YamlDataKey = `yaml::data`

// YamlLookupKey is a LookupKey function that uses the YamlData DataHash function to find the data and caches the result.
// It is mainly intended for testing purposes but can also be used as a complete replacement of a Configured hiera
// setup.
func YamlLookupKey(c hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
	data, ok := c.CachedValue(YamlDataKey)
	if !ok {
		data = YamlData(c, options)
		c.Cache(YamlDataKey, data)
	}
	hash, _ := data.(px.OrderedMap)
	v, ok := hash.Get4(key)
	if !ok {
		v = nil
	}
	return v
}
