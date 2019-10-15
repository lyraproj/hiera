package provider

import (
	"bytes"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/serialization"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func JSONData(c hieraapi.ServerContext) px.OrderedMap {
	pv := c.Option(`path`)
	if pv == nil {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `path`}))
	}
	path := pv.String()
	if bin, ok := types.BinaryFromFile2(path); ok {
		rdr := bytes.NewBuffer(bin.Bytes())
		vc := px.NewCollector()
		serialization.JsonToData(path, rdr, vc)
		v := vc.Value()
		if data, ok := v.(px.OrderedMap); ok {
			return data
		}
		panic(px.Error(hieraapi.JSONNOtHash, issue.H{`path`: path}))
	}
	return px.EmptyMap
}
