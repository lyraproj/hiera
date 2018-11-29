package provider

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-hiera/impl"
	"github.com/puppetlabs/go-hiera/lookup"
	"github.com/puppetlabs/go-issues/issue"
)

var YamlDataKey = `yaml::data`

// Environment provider performs a lookup in the current environment. The key can either be just
// "env" in which case all current environment variables will be returned as an OrderedMap, or
// prefixed with "env::" in which case the rest of the key is interpreted as the environment variable
// to look for.
func Yaml(c lookup.ProviderContext, key string, options map[string]eval.Value) (eval.Value, bool) {
	data, ok := c.CachedValue(YamlDataKey)
	if !ok {
		if v, ok := options[`path`]; ok {
			path := v.String()
			if bin, ok := types.BinaryFromFile2(c.Invocation(), path); ok {
				data = impl.UnmarshalYaml(c.Invocation(), bin.Bytes())
				if _, ok := data.(eval.OrderedMap); !ok {
					panic(eval.Error(impl.HIERA_YAML_NOT_HASH, issue.H{`path`: path}))
				}
			} else {
				// File not found. This is OK but yields an empty map
				data = eval.EMPTY_MAP
			}
			c.Cache(YamlDataKey, data)
		} else {
			panic(eval.Error(impl.HIERA_MISSING_REQUIRED_OPTION, issue.H{`option`: `path`}))
		}
	}
	hash, _ := data.(eval.OrderedMap)
	return hash.Get4(key)
}

