package examples_test

import (
	"context"
	"testing"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/provider"
	sdk "github.com/lyraproj/hierasdk/hiera"
)

// customLK is a custom lookup key function that just returns from the options that is passed to it
func customLK(hc sdk.ProviderContext, key string) dgo.Value {
	return hc.Option(key)
}

// TestCustomLK shows how to provide an in-process lookup function to Hiera using the api.HieraFunctions
// configuration option.
func TestCustomLK(t *testing.T) {
	// Provide custom functions in a dgo.Map so that they can be declared in the hiera configuration file. The function
	// signature must conform to the declared type (data_dig, data_hash, or lookup_key).
	//
	// The exact function signatures are defined in the hierasdk module as hiera.DataDig, hiera.DataHash, and
	// hiera.LookupKey.
	customFunctions := vf.Map(`customLK`, customLK)

	configOptions := vf.Map(
		api.HieraRoot, `testdata`,
		api.HieraConfigFileName, `custom.yaml`,
		api.HieraFunctions, customFunctions)

	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `a`, nil, nil)
		if result == nil || `option a` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}
