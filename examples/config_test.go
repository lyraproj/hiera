package examples_test

import (
	"context"
	"os"
	"testing"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/provider"
	sdk "github.com/lyraproj/hierasdk/hiera"
)

// sayHello is a very simple "lookup_key" function that just returns the result of concatenating
// the key with the string " world".
func sayHello(_ sdk.ProviderContext, key string) dgo.Value {
	return vf.String(key + ` world`)
}

/*
 Hiera will always use a single "lookup_key" function at the very top, henceforth referred to as the "top level
 function". This function determines the hierarchy (or lack thereof) that Hiera uses. This file explains the three such
 functions: the sayHello example function, the MuxLookupKey which aggregates other lookup_key functions, and the
 ConfigLookupKey which sets up a hierarchy defined in a yaml configuration.
*/

// TestConfig_hardwired utilizes Hiera in the simplest way possible. No configuration file and no options. Just
// a function performing a lookup of a key. In other words, this single function is the entire hierarchy.
func TestConfig_hardwired(t *testing.T) {
	// Use the hiera.DoWithParent to initialize a Hiera Session with the sayHello as the top-level function and
	// perform a lookup.
	//
	// The DoWithParent is meant to be called once and the created context can then be used for any number of lookups that
	// uses the same configuration. The session's life-cycle can be compared to the compiler's life-cycle in puppet.
	hiera.DoWithParent(context.Background(), sayHello, nil, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `hello world` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

// TestHelloWorld_semiHardWired uses the "lookup_key" function MuxLookupKey. This function use the configuration
// option LookupKeyFunctions where it expects to find a slice of "lookup_key" functions to use. Those functions form
// top level hierarchy that is configurable from code. Very useful if you for instance want to introduce different
// lookup layers such as "global", "environment", and "module" or in other ways build a complex lookup hierarchy that
// service that goes beyond what can be defined in the yaml configuration.
func TestConfig_semiHardWired(t *testing.T) {
	// Create options valid for this Hiera session.
	// The LookupProvidersKey stores a go slice of Hiera "lookup_key" functions that serve as the top level functions.
	configOptions := vf.Map(provider.LookupKeyFunctions, []sdk.LookupKey{sayHello})

	// Initialize a Hiera session with the MuxLookupKey as the top-level function and perform a lookup and
	// the created configOptions.
	hiera.DoWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `hello world` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

/*
 The remaining tests in this file use the ConfigLookupKey provider. This provider will consult the configuration
 options HieraRoot, HieraConfigFileName, and HieraConfig to determine the path of the configuration file. Use of
 HieraConfig is mutually exclusive with HieraRoot and HieraConfigFileName.

 HieraRoot will default to the current working directory.

 HieraConfigFileName will default to "hiera.yaml".

 If the HieraRoot or HieraConfig are relative paths, they will be considered relative to the current directory.

 The HieraConfigFileName must be relative to the HieraRoot.
*/

// TestHelloWorld_yamlConfig uses the "lookup_key" function ConfigLookupKey and HieraRoot. The ConfigLookupKey is
// the most commonly used top-level function in Hiera. It finds a yaml configuration on disk and then configures
// everything according to the hierarchy specified in that file.
func TestHelloWorld_yamlConfig(t *testing.T) {
	configOptions := vf.Map(api.HieraRoot, `testdata`)

	// Initialize a Hiera session with the ConfigLookupKey as the top-level function configured using the configOptions.
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `yaml data says hello` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

// TestHelloWorld_explicitYamlConfig is similar to TestHelloWorld_yamlConfig but uses HieraConfig
// option to explicitly define the file to use.
func TestHelloWorld_explicitYamlConfig(t *testing.T) {
	configOptions := vf.Map(api.HieraConfig, `testdata/hiera.yaml`)

	// Initialize a Hiera session with the ConfigLookupKey as the top-level function configured using the configOptions.
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `yaml data says hello` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

// TestHelloWorld_explicitYamlConfigEnvironment is similar to TestHelloWorld_yamlConfig but uses HIERA_CONFIGFILE
// environment variable to explicitly define the file to use.
func TestHelloWorld_explicitYamlConfigEnvironment(t *testing.T) {
	configOptions := vf.Map(api.HieraRoot, `testdata`)
	_ = os.Setenv("HIERA_CONFIGFILE", "hiera_env.yaml")

	// Initialize a Hiera session with the ConfigLookupKey as the top-level function configured using the configOptions.
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `yaml data says hello` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

// TestHelloWorld_explicitYamlConfigFile is similar to TestHelloWorld_yamlConfig but uses a combination of
// HieraRoot and HieraConfigFileName option to find the yaml configuration file.
func TestHelloWorld_explicitYamlConfigFile(t *testing.T) {
	configOptions := vf.Map(
		api.HieraRoot, `testdata`,
		api.HieraConfigFileName, `special.yaml`)

	// Initialize a Hiera session with the ConfigLookupKey as the top-level function configured using the configOptions.
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `yaml special data says hello` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

// TestHelloWorld_yamlAndSemiHardWired uses the MuxLookupKey to inject two lookup_key functions. The ConfigLookupKey
// that consults the yaml config and the sayHello.
func TestHelloWorld_yamlAndSemiHardWired(t *testing.T) {
	configOptions := vf.Map(
		provider.LookupKeyFunctions, []sdk.LookupKey{provider.ConfigLookupKey, sayHello},
		api.HieraRoot, `testdata`)

	// Initialize a Hiera session with the MuxLookupKey as the top-level function configured using the configOptions.
	hiera.DoWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(hs api.Session) {
		// A lookup of just "hello" should hit the first provider, the ConfigLookupKey.
		result := hiera.Lookup(hs.Invocation(nil, nil), `hello`, nil, nil)
		if result == nil || `yaml data says hello` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// A lookup of "howdy" is not found using the yaml configuration, so it hits the second provider, the sayHello.
		result = hiera.Lookup(hs.Invocation(nil, nil), `howdy`, nil, nil)
		if result == nil || `howdy world` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}
