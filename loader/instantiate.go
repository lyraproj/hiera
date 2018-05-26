package loader

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/loader"
	"gopkg.in/yaml.v2"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-evaluator/types"
)

func InstantiateHieraConfig(c eval.Context, loader loader.ContentProvidingLoader, tn eval.TypedName, sources []string) {
	source := sources[0]
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal([]byte(loader.GetContent(c, source)), &ms)
	if err != nil {
		panic(eval.Error(c, eval.EVAL_PARSE_ERROR, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	cfgType := c.ParseType2(`Hiera::Config`)
	configHash := eval.AssertInstance(c, func() string { return source }, cfgType, eval.Wrap2(c, ms)).(*types.HashValue)
	configHash.Len()
}

