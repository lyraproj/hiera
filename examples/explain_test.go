package examples_test

import (
	"context"
	"testing"

	"github.com/lyraproj/hiera/explain"

	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/internal"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// TestExplain shows how to provide an explainer to the invocation used when performing a lookup
// and to extract its result.
func TestExplain(t *testing.T) {
	configOptions := map[string]px.Value{hieraapi.HieraRoot: types.WrapString(`testdata`)}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(c px.Context) {
		// The scope type from the scope_test.go file is reused.
		s := scope{
			`a`: `the "a" string`,
			`b`: 42,
			`c`: map[string]int{`x`: 10, `y`: 20}}

		// Create an explainer that excludes the lookups of lookup_options
		explainer := explain.NewExplainer(false, false)

		// Perform a lookup where the scope and explainer are included in the invocation
		result := hiera.Lookup(internal.NewInvocation(c, s, explainer), `ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		expectedExplanation := `Searching for "ipl_c"
  Merge strategy "first found strategy"
    data_hash function 'yaml_data'
      Path "testdata/data.yaml"
        Original path: "data.yaml"
        path not found
    data_hash function 'yaml_data'
      Path "testdata/data/common.yaml"
        Original path: "common.yaml"
        Interpolation on "x = %{c.x}, y = %{c.y}"
          Sub key: "x"
            Found key: "x" value: 10
          Sub key: "y"
            Found key: "y" value: 20
        Found key: "ipl_c" value: 'x = 10, y = 20'
    Merged result: 'x = 10, y = 20'`

		actualExplanation := explainer.String()
		if expectedExplanation != explainer.String() {
			t.Fatalf("expected explanation `%s` does not match actual `%s`", expectedExplanation, actualExplanation)
		}
	})
}

// TestExplain_withOptions shows how to configure the Explainer to include the lookup of the lookup_options
func TestExplain_withOptions(t *testing.T) {
	configOptions := map[string]px.Value{hieraapi.HieraRoot: types.WrapString(`testdata`)}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(c px.Context) {
		s := scope{
			`a`: `the "a" string`,
			`b`: 42,
			`c`: map[string]int{`x`: 10, `y`: 20}}

		// Create an explainer that includes lookups of lookup_options
		explainer := explain.NewExplainer(true, false)

		// Perform a lookup where the scope and explainer are included in the invocation
		result := hiera.Lookup(internal.NewInvocation(c, s, explainer), `ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		expectedExplanation := `Searching for "lookup_options"
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "testdata/data.yaml"
        Original path: "data.yaml"
        path not found
    data_hash function 'yaml_data'
      Path "testdata/data/common.yaml"
        Original path: "common.yaml"
        No such key: "lookup_options"
Searching for "ipl_c"
  Merge strategy "first found strategy"
    data_hash function 'yaml_data'
      Path "testdata/data.yaml"
        Original path: "data.yaml"
        path not found
    data_hash function 'yaml_data'
      Path "testdata/data/common.yaml"
        Original path: "common.yaml"
        Interpolation on "x = %{c.x}, y = %{c.y}"
          Sub key: "x"
            Found key: "x" value: 10
          Sub key: "y"
            Found key: "y" value: 20
        Found key: "ipl_c" value: 'x = 10, y = 20'
    Merged result: 'x = 10, y = 20'`

		actualExplanation := explainer.String()
		if expectedExplanation != explainer.String() {
			t.Fatalf("expected explanation `%s` does not match actual `%s`", expectedExplanation, actualExplanation)
		}
	})
}

// TestExplain_withOnlyOptions shows how to configure the Explainer to only include the lookup of the lookup_options
// and exclude the lookup of the actual value
func TestExplain_withOnlyOptions(t *testing.T) {
	configOptions := map[string]px.Value{hieraapi.HieraRoot: types.WrapString(`testdata`)}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(c px.Context) {
		s := scope{
			`a`: `the "a" string`,
			`b`: 42,
			`c`: map[string]int{`x`: 10, `y`: 20}}

		// Create an explainer that only includes the lookups of lookup_options and excludes everything else
		explainer := explain.NewExplainer(true, true)

		// Perform a lookup where the scope and explainer are included in the invocation
		result := hiera.Lookup(internal.NewInvocation(c, s, explainer), `ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		expectedExplanation := `Searching for "lookup_options"
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "testdata/data.yaml"
        Original path: "data.yaml"
        path not found
    data_hash function 'yaml_data'
      Path "testdata/data/common.yaml"
        Original path: "common.yaml"
        No such key: "lookup_options"`

		actualExplanation := explainer.String()
		if expectedExplanation != explainer.String() {
			t.Fatalf("expected explanation `%s` does not match actual `%s`", expectedExplanation, actualExplanation)
		}
	})
}
