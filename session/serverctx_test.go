package session_test

import (
	"context"
	"fmt"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	sdk "github.com/lyraproj/hierasdk/hiera"
)

func ExampleServerContext_CachedValue() {
	cachingProvider := func(pc sdk.ProviderContext, key string) dgo.Value {
		ic := pc.(api.ServerContext)
		if v, ok := ic.CachedValue(key); ok {
			fmt.Printf("Returning cached value for %s\n", key)
			return v
		}
		fmt.Printf("Creating and caching value for %s\n", key)
		v := ic.Interpolate(vf.String(fmt.Sprintf("value for %%{%s}", key)))
		ic.Cache(key, v)
		return v
	}

	hiera.DoWithParent(context.Background(), cachingProvider, nil, func(hs api.Session) {
		s := map[string]interface{}{
			`a`: `scope 'a'`,
			`b`: `scope 'b'`,
		}
		ic := hs.Invocation(s, nil)
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
