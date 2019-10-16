package provider

import (
	"io/ioutil"
	"os"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/streamer"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
)

func JSONData(ctx hiera.ProviderContext) dgo.Map {
	pv := ctx.Option(`path`)
	if pv == nil {
		panic(hieraapi.MissingRequiredOption(`path`))
	}
	path := pv.String()
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vf.Map()
		}
		panic(err)
	}
	v := streamer.UnmarshalJSON(bs, nil)
	if data, ok := v.(dgo.Map); ok {
		return data
	}
	panic(hieraapi.JSONNOtHash(path))
}
