package loader

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/loader"
	"github.com/lyraproj/puppet-evaluator/types"
	"gopkg.in/yaml.v2"
)

func InstantiateHieraConfig(c eval.Context, loader loader.ContentProvidingLoader, tn eval.TypedName, sources []string) {
	source := sources[0]
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal([]byte(loader.GetContent(c, source)), &ms)
	if err != nil {
		panic(eval.Error(eval.EVAL_PARSE_ERROR, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	cfgType := c.ParseType2(`Hiera::Config`)
	configHash := eval.AssertInstance(func() string { return source }, cfgType, eval.Wrap(c, ms)).(*types.HashValue)
	configHash.Len()
}
