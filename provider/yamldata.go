package provider

import (
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
)

func YamlData(ctx hieraapi.ServerContext) px.OrderedMap {
	pv := ctx.Option(`path`)
	if pv == nil {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `path`}))
	}
	path := pv.String()
	if bin, ok := types.BinaryFromFile2(path); ok {
		v := yaml.Unmarshal(ctx.(hieraapi.ServerContext).Invocation(), bin.Bytes())
		if data, ok := v.(px.OrderedMap); ok {
			return data
		}
		panic(px.Error(hieraapi.YamlNotHash, issue.H{`path`: path}))
	}
	return px.EmptyMap
}
