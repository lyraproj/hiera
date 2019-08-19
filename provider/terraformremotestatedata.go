package provider

import (
	"encoding/json"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"

	backendInit "github.com/hashicorp/terraform/backend/init"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func TerraformRemoteStateData(ctx hieraapi.ProviderContext, options map[string]px.Value) px.OrderedMap {
	backendName, ok := options[`backend`]
	if !ok {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `backend`}))
	}
	backend := backendName.String()
	workspaceName, ok := options[`workspace`]
	var workspace string
	if !ok {
		workspace = "default"
	} else {
		workspace = workspaceName.String()
	}
	configMap, ok := options[`config`]
	if !ok {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `config`}))
	}
	var conf map[string]cty.Value
	if cm, ok := configMap.(px.OrderedMap); ok {
		cm.EachPair(func(k, v px.Value) {
			conf[k.String()] = cty.StringVal(v.String())
		})
	}
	config := cty.ObjectVal(conf)
	backendInit.Init(nil)
	f := backendInit.Backend(backend)
	if f == nil {
		panic("unknown backend type")
	}
	b := f()
	newVal, _ := b.PrepareConfig(config)
	config = newVal
	_ = b.Configure(config)
	state, _ := b.StateMgr(workspace)
	_ = state.RefreshState()
	remoteState := state.State()
	mod := remoteState.RootModule()
	outputjson := make(map[string]interface{})
	for k, os := range mod.OutputValues {
		ctyjson, _ := ctyjson.Marshal(os.Value, os.Value.Type())
		outputjson[k] = json.RawMessage(ctyjson)
	}
	hsh := px.Wrap(nil, outputjson)
	return hsh.(px.OrderedMap)
}
