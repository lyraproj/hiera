package provider

import (
	"github.com/lyraproj/hiera/impl"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
)

var YamlDataKey = `yaml::data`

// Environment provider performs a lookup in the current environment. The key can either be just
// "env" in which case all current environment variables will be returned as an OrderedMap, or
// prefixed with "env::" in which case the rest of the key is interpreted as the environment variable
// to look for.
func Yaml(c lookup.ProviderContext, key string, options map[string]px.Value) (px.Value, bool) {
	data, ok := c.CachedValue(YamlDataKey)
	if !ok {
		data = impl.YamlData(c, options)
		c.Cache(YamlDataKey, data)
	}
	hash, _ := data.(px.OrderedMap)
	return hash.Get4(key)
}
