package impl

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/issue/issue"
	"gopkg.in/yaml.v2"
)

func UnmarshalYaml(c eval.Context, data []byte) eval.Value {
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal([]byte(data), &ms)
	if err != nil {
		var itm interface{}
		err2 := yaml.Unmarshal([]byte(data), &itm)
		if err2 != nil {
			panic(eval.Error(eval.EVAL_PARSE_ERROR, issue.H{`language`: `YAML`, `detail`: err.Error()}))
		}
		return wrapValue(c, itm)
	}
	return wrapSlice(c, ms)
}

func wrapSlice(c eval.Context, ms yaml.MapSlice) eval.Value {
	es := make([]*types.HashEntry, len(ms))
	for i, me := range ms {
		es[i] = types.WrapHashEntry(wrapValue(c, me.Key), wrapValue(c, me.Value))
	}
	return types.WrapHash(es)
}

func wrapValue(c eval.Context, v interface{}) eval.Value {
	switch v.(type) {
	case yaml.MapSlice:
		return wrapSlice(c, v.(yaml.MapSlice))
	case []interface{}:
		ys := v.([]interface{})
		vs := make([]eval.Value, len(ys))
		for i, y := range ys {
			vs[i] = wrapValue(c, y)
		}
		return types.WrapValues(vs)
	default:
		return eval.Wrap(c, v)
	}
}