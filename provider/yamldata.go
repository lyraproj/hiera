package provider

import (
	"io/ioutil"
	"os"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/dgoyaml/yaml"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hierasdk/hiera"
)

// YamlData is a data_hash provider that reads a YAML hash from a file and returns it as a Map
func YamlData(ctx hiera.ProviderContext) dgo.Map {
	pv := ctx.Option(`path`)
	if pv == nil {
		panic(api.MissingRequiredOption(`path`))
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
	panic(api.YamlNotHash(path))
}
