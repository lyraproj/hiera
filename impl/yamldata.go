package impl

import (
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
)

func YamlData(ctx lookup.ProviderContext, options map[string]px.Value) px.OrderedMap {
	pv, ok := options[`path`]
	if !ok {
		panic(px.Error(MissingRequiredOption, issue.H{`option`: `path`}))
	}
	path := pv.String()
	var bin *types.Binary
	if bin, ok = types.BinaryFromFile2(path); ok {
		v := yaml.Unmarshal(ctx.(*providerCtx).invocation, bin.Bytes())
		if data, ok := v.(px.OrderedMap); ok {
			return data
		}
		panic(px.Error(YamlNotHash, issue.H{`path`: path}))
	}
	return px.EmptyMap
}
