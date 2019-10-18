package examples_test

import (
	"context"
	"testing"

	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// scope is a px.Keyed implementation in its simplest form
type scope map[string]interface{}

func (s scope) Get(key px.Value) (px.Value, bool) {
	if v, ok := s[key.String()]; ok {
		return px.Wrap(nil, v), true
	}
	return nil, false
}

// TestScope shows how to provide a "scope" to the invocation.
func TestScope(t *testing.T) {
	configOptions := map[string]px.Value{hieraapi.HieraRoot: types.WrapString(`testdata`)}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(c px.Context) {
		// Our scope is just a map[string]interface{} and can hold any arbitrary data
		s := scope{
			`a`: `the "a" string`,
			`b`: 42,
			`c`: map[string]int{`x`: 10, `y`: 20}}

		// The value of key "ipl_a" uses interpolation and includes %{a} in the result
		result := hiera.Lookup(hiera.NewInvocation(c, s, nil), `ipl_a`, nil, nil)
		if result == nil || `interpolate <the "a" string>` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// The value of key "ipl_c" is "x = %{c.x}, y = %{c.y}"
		result = hiera.Lookup(hiera.NewInvocation(c, s, nil), `ipl_c`, nil, nil)
		if result == nil || `x = 10, y = 20` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}
