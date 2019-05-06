package hieraapi_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lyraproj/hiera/hieraapi"

	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraimpl"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

var options map[string]px.Value

func init() {
	options = map[string]px.Value{`path`: types.WrapString(`./testdata/sample_data.yaml`)}
}

func ExampleLookup_first() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `first`, nil, nil))
	})
	// Output: value of first
}

func ExampleLookup_dottedInt() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `array.1`, nil, nil))
	})
	// Output: two
}

func ExampleLookup_dottedMix() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `hash.array.1`, nil, nil))
	})
	// Output: value of first
}

func ExampleLookup_interpolate() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `second`, nil, nil))
	})
	// Output: includes 'value of first'
}

func ExampleLookup_interpolateScope() {
	pcore.DoWithParent(context.Background(), func(c px.Context) {
		s := types.WrapStringToInterfaceMap(c, issue.H{
			`world`: `cruel world`,
		})
		hiera.DoWithParent(c, provider.YamlLookupKey, options, func(c px.Context) {
			fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `ipScope`, nil, nil))
			fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `ipScope2`, nil, nil))
		})
	})
	// Output:
	// hello cruel world
	// hello cruel world
}

func ExampleLookup_interpolateEmpty() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		s := px.EmptyMap
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `empty1`, nil, nil))
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `empty2`, nil, nil))
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `empty3`, nil, nil))
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `empty4`, nil, nil))
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `empty5`, nil, nil))
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, s), `empty6`, nil, nil))
	})
	// Output:
	// StartEnd
	// StartEnd
	// StartEnd
	// StartEnd
	// StartEnd
	// StartEnd
}

func printErr(e error) {
	s := e.Error()
	if ix := strings.Index(s, ` (file: `); ix > 0 {
		s = s[0:ix]
	}
	fmt.Println(s)
}

func ExampleLookup_interpolateLiteral() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `ipLiteral`, nil, options))
	})
	// Output: some literal text
}

func ExampleLookup_interpolateAlias() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		v := hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `ipAlias`, nil, options)
		fmt.Printf(`%s %s`, px.GenericValueType(v), v)
	})
	// Output: Array[Enum] ['one', 'two', 'three']
}

func ExampleLookup_interpolateBadAlias() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `ipBadAlias`, nil, options)
		return nil
	}))
	// Output: 'alias' interpolation is only permitted if the expression is equal to the entire string
}

func ExampleLookup_interpolateBadFunction() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `ipBad`, nil, options)
		return nil
	}))
	// Output: Unknown interpolation method 'bad'
}

func ExampleLookup_notFoundWithoutDefault() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `nonexistent`, nil, options)
		return nil
	}))
	// Output: lookup() did not find a value for the name 'nonexistent'
}

func ExampleLookup_notFoundDflt() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `nonexistent`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedIdx() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `array.3`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedMix() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `hash.float`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_badStringDig() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `hash.int.v`, nil, options)
		return nil
	}))
	// Output: lookup() Got Integer when a hash-like object was expected to access value using 'v' from key 'hash.int.v'
}

func ExampleLookup_badIntDig() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `hash.3`, nil, options)
		return nil
	}))
	// Output: lookup() Got Hash[Enum, Data] when a hash-like object was expected to access value using '3' from key 'hash.3'
}

func ExampleLookup2_findFirst() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup2(hieraimpl.NewInvocation(c, px.EmptyMap), []string{`first`, `second`}, types.DefaultAnyType(), nil, nil, nil, options, nil))
	})
	// Output: value of first
}

func ExampleLookup2_findSecond() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup2(hieraimpl.NewInvocation(c, px.EmptyMap), []string{`non existing`, `second`}, types.DefaultAnyType(), nil, nil, nil, options, nil))
	})
	// Output: includes 'value of first'
}

func ExampleLookup2_notFoundWithoutDflt() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup2(hieraimpl.NewInvocation(c, px.EmptyMap), []string{`non existing`, `not there`}, types.DefaultAnyType(), nil, nil, nil, options, nil)
		return nil
	}))
	// Output: lookup() did not find a value for any of the names [non existing, not there]
}

func ExampleLookup2_notFoundDflt() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup2(hieraimpl.NewInvocation(c, px.EmptyMap), []string{`non existing`, `not there`}, types.DefaultAnyType(), types.WrapString(`default value`), nil, nil, options, nil))
	})
	// Output: default value
}

func ExampleProviderContext_CachedValue() {

	cachingProvider := func(ic hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
		if v, ok := ic.CachedValue(key); ok {
			fmt.Printf("Returning cached value for %s\n", key)
			return v
		}
		fmt.Printf("Creating and caching value for %s\n", key)
		v := ic.Interpolate(types.WrapString(fmt.Sprintf("generated value for %%{%s}", key)))
		ic.Cache(key, v)
		return v
	}

	hiera.DoWithParent(context.Background(), cachingProvider, map[string]px.Value{}, func(c px.Context) {
		s := types.WrapStringToInterfaceMap(c, map[string]interface{}{
			`a`: `scope 'a'`,
			`b`: `scope 'b'`,
		})
		ic := hieraimpl.NewInvocation(c, s)
		fmt.Println(hiera.Lookup(ic, `a`, nil, nil))
		fmt.Println(hiera.Lookup(ic, `b`, nil, nil))
		fmt.Println(hiera.Lookup(ic, `a`, nil, nil))
		fmt.Println(hiera.Lookup(ic, `b`, nil, nil))
	})
	// Output:
	// Creating and caching value for a
	// generated value for scope 'a'
	// Creating and caching value for b
	// generated value for scope 'b'
	// generated value for scope 'a'
	// generated value for scope 'b'
}

func ExampleLookup_dottedStringInt() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		v := hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `hash.array.0`, nil, options)
		fmt.Println(v)
	})
	// Output: two
}

func ExampleLookup_mapProvider() {
	sampleData := map[string]string{
		`a`: `value of a`,
		`b`: `value of b`}

	tp := func(ic hieraapi.ProviderContext, key string, _ map[string]px.Value) px.Value {
		if v, ok := sampleData[key]; ok {
			return types.WrapString(v)
		}
		return nil
	}

	hiera.DoWithParent(context.Background(), tp, map[string]px.Value{}, func(c px.Context) {
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `a`, nil, nil))
		fmt.Println(hiera.Lookup(hieraimpl.NewInvocation(c, px.EmptyMap), `b`, nil, nil))
	})

	// Output:
	// value of a
	// value of b
}
