package examples_test

import (
	"context"
	"testing"

	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/provider"
)

// TestScope shows how to provide a "scope" to the invocation.
func TestScope(t *testing.T) {
	configOptions := vf.Map(api.HieraRoot, `testdata`)
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		// Our scope is just a map[string]interface{} and can hold any arbitrary data
		s := map[string]interface{}{
			`a`: `the "a" string`,
			`b`: 42,
			`c`: map[string]int{`x`: 10, `y`: 20}}

		// The value of key "ipl_a" uses interpolation and includes %{a} in the result
		result := hiera.Lookup(hs.Invocation(s, nil), `ipl_a`, nil, nil)
		if result == nil || `interpolate <the "a" string>` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// The value of key "ipl_c" is "x = %{c.x}, y = %{c.y}"
		result = hiera.Lookup(hs.Invocation(s, nil), `ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}
