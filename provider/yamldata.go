package provider

import (
	"io/ioutil"
	"os"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/dgoyaml/yaml"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
)

func YamlData(ctx hiera.ProviderContext) dgo.Map {
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
	v, err := yaml.Unmarshal(bs)
	if err != nil {
		panic(err)
	}
	if data, ok := v.(dgo.Map); ok {
		return data
	}
	panic(hieraapi.YamlNotHash(path))
}
