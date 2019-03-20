package provider

import (
	"github.com/lyraproj/hiera/impl"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
)

var YamlDataKey = `yaml::data`

// Environment provider performs a lookup in the current environment. The key can either be just
// "env" in which case all current environment variables will be returned as an OrderedMap, or
// prefixed with "env::" in which case the rest of the key is interpreted as the environment variable
// to look for.
func Yaml(c lookup.ProviderContext, key string, options map[string]px.Value) (px.Value, bool) {
	data, ok := c.CachedValue(YamlDataKey)
	if !ok {
		if v, ok := options[`path`]; ok {
			path := v.String()
			if bin, ok := types.BinaryFromFile2(path); ok {
				data = yaml.Unmarshal(c.Invocation(), bin.Bytes())
				if _, ok := data.(px.OrderedMap); !ok {
					panic(px.Error(impl.HieraYamlNotHash, issue.H{`path`: path}))
				}
			} else {
				// File not found. This is OK but yields an empty map
				data = px.EmptyMap
			}
			c.Cache(YamlDataKey, data)
		} else {
			panic(px.Error(impl.HieraMissingRequiredOption, issue.H{`option`: `path`}))
		}
	}
	hash, _ := data.(px.OrderedMap)
	return hash.Get4(key)
}
