package provider

import (
	"bytes"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/serialization"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func JsonData(_ hieraapi.ProviderContext, options map[string]px.Value) px.OrderedMap {
	pv, ok := options[`path`]
	if !ok {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `path`}))
	}
	path := pv.String()
	var bin *types.Binary
	if bin, ok = types.BinaryFromFile2(path); ok {
		rdr := bytes.NewBuffer(bin.Bytes())
		vc := px.NewCollector()
		serialization.JsonToData(path, rdr, vc)
		v := vc.Value()
		if data, ok := v.(px.OrderedMap); ok {
			return data
		}
		panic(px.Error(hieraapi.JsonNotHash, issue.H{`path`: path}))
	}
	return px.EmptyMap
}
