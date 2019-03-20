package loader

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/loader"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"gopkg.in/yaml.v2"
)

func InstantiateHieraConfig(c px.Context, loader loader.ContentProvidingLoader, tn px.TypedName, sources []string) {
	source := sources[0]
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal([]byte(loader.GetContent(c, source)), &ms)
	if err != nil {
		panic(px.Error(px.ParseError, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	cfgType := c.ParseType(`Hiera::Config`)
	configHash := px.AssertInstance(func() string { return source }, cfgType, px.Wrap(c, ms)).(*types.Hash)
	configHash.Len()
}
