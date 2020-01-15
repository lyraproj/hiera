package examples

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/provider"
	sdk "github.com/lyraproj/hierasdk/hiera"
)

// TestHelloWorld_globalAndModules uses the MuxLookupKey to inject two lookup_key functions. The ConfigLookupKey
// that consults the yaml config and the ModuleLookupKey that consults the provider.ModulePath to find modules
// that in turn contains additional configuration and data.
func TestHelloWorld_globalAndModules(t *testing.T) {
	configOptions := vf.Map(
		provider.LookupKeyFunctions, []sdk.LookupKey{provider.ConfigLookupKey, provider.ModuleLookupKey},
		api.HieraRoot, `testdata`,
		provider.ModulePath, filepath.Join(`testdata`, `modules`))

	// Initialize a Hiera session with the MuxLookupKey as the top-level function configured using the configOptions.
	hiera.DoWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(hs api.Session) {
		// A lookup of just "hello" should hit the first provider, the ConfigLookupKey.
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `yaml data says hello` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// A lookup of "one::a" is found in the module "one"
		result = hiera.Lookup(hs.Invocation(nil, nil), `one::a`, nil, nil)
		if result == nil || `value of one::a` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// A lookup of "one::merge" is found in the ConfigLookupKey provider and in module "one". The lookup_options
		// declared in the module "one" stipulates a deep merge.
		result = hiera.Lookup(hs.Invocation(nil, nil), `one::merge`, nil, nil)
		if result == nil || `{"a":"value of one::merge a","b":"value of one::merge b"}` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// A lookup of "three::a" will not find a value because the "three" directory does not contain a hiera.yaml
		result = hiera.Lookup(hs.Invocation(nil, nil), `three::a`, nil, nil)
		if result != nil {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

// TestHelloWorld_globalAndModules_nonExistentPath uses a path that doesn't appoint a directory.
func TestHelloWorld_globalAndModules_nonExistentPath(t *testing.T) {
	configOptions := vf.Map(
		provider.LookupKeyFunctions, []sdk.LookupKey{provider.ConfigLookupKey, provider.ModuleLookupKey},
		api.HieraRoot, `testdata`,
		provider.ModulePath, filepath.Join(`testdata`, `nomodules`))

	hiera.DoWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `one::a`, nil, nil)
		if result != nil {
			t.Fatalf("unexpected result %v", result)
		}
	})
}
