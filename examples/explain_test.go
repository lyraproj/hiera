package examples_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lyraproj/dgo/vf"

	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/explain"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/provider"
)

// TestExplain shows how to provide an explainer to the invocation used when performing a lookup
// and to extract its result.
func TestExplain(t *testing.T) {
	configOptions := vf.Map(
		api.HieraRoot, `testdata`,
		provider.ModulePath, filepath.Join(`testdata`, `modules`))

	hiera.DoWithParent(context.Background(), provider.ModuleLookupKey, configOptions, func(hs api.Session) {
		// The scope type from the scope_test.go file is reused.
		s := map[string]interface{}{`c`: map[string]int{`x`: 10, `y`: 20}}

		// Create an explainer that excludes the lookups of lookup_options
		explainer := explain.NewExplainer(false, false)

		// Perform a lookup where the scope and explainer are included in the invocation
		result := hiera.Lookup(hs.Invocation(s, explainer), `one::ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		expectedExplanation := filepath.FromSlash(`Searching for "one::ipl_c"
  data_hash function 'yaml_data'
    Path "testdata/modules/one/data.yaml"
      Original path: "data.yaml"
      path not found
  data_hash function 'yaml_data'
    Path "testdata/modules/one/data/common.yaml"
      Original path: "common.yaml"
      Interpolation on "x = %{c.x}, y = %{c.y}"
        Sub key: "x"
          Found key: "x" value: 10
        Sub key: "y"
          Found key: "y" value: 20
      Found key: "one::ipl_c" value: "x = 10, y = 20"`)

		actualExplanation := explainer.String()
		if expectedExplanation != actualExplanation {
			t.Fatalf("expected explanation `%s` does not match actual `%s`", expectedExplanation, actualExplanation)
		}
	})
}

// TestExplain_withOptions shows how to configure the Explainer to include the lookup of the lookup_options
func TestExplain_withOptions(t *testing.T) {
	configOptions := map[string]string{api.HieraRoot: `testdata`}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		s := map[string]interface{}{
			`a`: `the "a" string`,
			`b`: 42,
			`c`: map[string]int{`x`: 10, `y`: 20}}

		// Create an explainer that includes lookups of lookup_options
		explainer := explain.NewExplainer(true, false)

		// Perform a lookup where the scope and explainer are included in the invocation
		result := hiera.Lookup(hs.Invocation(s, explainer), `ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		expectedExplanation := filepath.FromSlash(`Searching for "lookup_options"
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
      Found key: "ipl_c" value: "x = 10, y = 20"`)

		actualExplanation := explainer.String()
		if expectedExplanation != explainer.String() {
			t.Fatalf("expected explanation `%s` does not match actual `%s`", expectedExplanation, actualExplanation)
		}
	})
}

// TestExplain_withOnlyOptions shows how to configure the Explainer to only include the lookup of the lookup_options
// and exclude the lookup of the actual value
func TestExplain_withOnlyOptions(t *testing.T) {
	configOptions := map[string]string{api.HieraRoot: `testdata`}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		s := map[string]interface{}{
			`a`: `the "a" string`,
			`b`: 42,
			`c`: map[string]int{`x`: 10, `y`: 20}}

		// Create an explainer that only includes the lookups of lookup_options and excludes everything else
		explainer := explain.NewExplainer(true, true)

		// Perform a lookup where the scope and explainer are included in the invocation
		result := hiera.Lookup(hs.Invocation(s, explainer), `ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		expectedExplanation := filepath.FromSlash(`Searching for "lookup_options"
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "testdata/data.yaml"
        Original path: "data.yaml"
        path not found
    data_hash function 'yaml_data'
      Path "testdata/data/common.yaml"
        Original path: "common.yaml"
        No such key: "lookup_options"`)

		actualExplanation := explainer.String()
		if expectedExplanation != explainer.String() {
			t.Fatalf("expected explanation `%s` does not match actual `%s`", expectedExplanation, actualExplanation)
		}
	})
}
