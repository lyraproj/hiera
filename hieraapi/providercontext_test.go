package hieraapi_test

import (
	"context"
	"fmt"

	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func ExampleProviderContext_CachedValue() {

	cachingProvider := func(ic hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
		if v, ok := ic.CachedValue(key); ok {
			fmt.Printf("Returning cached value for %s\n", key)
			return v
		}
		fmt.Printf("Creating and caching value for %s\n", key)
		v := ic.Interpolate(types.WrapString(fmt.Sprintf("value for %%{%s}", key)))
		ic.Cache(key, v)
		return v
	}

	hiera.DoWithParent(context.Background(), cachingProvider, map[string]px.Value{}, func(c px.Context) {
		s := types.WrapStringToInterfaceMap(c, map[string]interface{}{
			`a`: `scope 'a'`,
			`b`: `scope 'b'`,
		})
		ic := hiera.NewInvocation(c, s, nil)
		fmt.Println(hiera.Lookup(ic, `a`, nil, nil))
		fmt.Println(hiera.Lookup(ic, `b`, nil, nil))
		fmt.Println(hiera.Lookup(ic, `a`, nil, nil))
		fmt.Println(hiera.Lookup(ic, `b`, nil, nil))
	})
	// Output:
	// Creating and caching value for a
	// value for scope 'a'
	// Creating and caching value for b
	// value for scope 'b'
	// Returning cached value for a
	// value for scope 'a'
	// Returning cached value for b
	// value for scope 'b'
}
