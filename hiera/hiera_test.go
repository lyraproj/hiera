package hiera_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	require "github.com/lyraproj/dgo/dgo_test"

	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/internal"
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
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `first`, nil, nil))
	})
	// Output: value of first
}

func TestLookup_dottedInt(t *testing.T) {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		require.Equal(t, `two`, hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `array.1`, nil, nil).String())
	})
}

func TestLookup_dottedMix(t *testing.T) {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		require.Equal(t, `value of first`,
			hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `hash.array.1`, nil, nil).String())
	})
}

func TestLookup_interpolate(t *testing.T) {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		require.Equal(t, `includes 'value of first'`,
			hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `second`, nil, nil).String())
	})
}

func ExampleLookup_interpolateScope() {
	pcore.DoWithParent(context.Background(), func(c px.Context) {
		s := types.WrapStringToInterfaceMap(c, issue.H{
			`world`: `cruel world`,
		})
		hiera.DoWithParent(c, provider.YamlLookupKey, options, func(c px.Context) {
			fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `ipScope`, nil, nil))
			fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `ipScope2`, nil, nil))
		})
	})
	// Output:
	// hello cruel world
	// hello cruel world
}

func ExampleLookup_interpolateEmpty() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		s := px.EmptyMap
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `empty1`, nil, nil))
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `empty2`, nil, nil))
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `empty3`, nil, nil))
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `empty4`, nil, nil))
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `empty5`, nil, nil))
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, s, nil), `empty6`, nil, nil))
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
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `ipLiteral`, nil, options))
	})
	// Output: some literal text
}

func ExampleLookup_interpolateAlias() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		v := hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `ipAlias`, nil, options)
		fmt.Printf(`%s %s`, px.GenericValueType(v), v)
	})
	// Output: Array[Enum] ['one', 'two', 'three']
}

func ExampleLookup_interpolateBadAlias() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `ipBadAlias`, nil, options)
		return nil
	}))
	// Output: 'alias' interpolation is only permitted if the expression is equal to the entire string
}

func ExampleLookup_interpolateBadFunction() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `ipBad`, nil, options)
		return nil
	}))
	// Output: Unknown interpolation method 'bad'
}

func ExampleLookup_notFoundWithoutDefault() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `nonexistent`, nil, options)
		return nil
	}))
	// Output: lookup() did not find a value for the name 'nonexistent'
}

func ExampleLookup_notFoundDflt() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `nonexistent`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedIdx() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `array.3`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedMix() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `hash.float`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_badStringDig() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `hash.int.v`, nil, options)
		return nil
	}))
	// Output: lookup() did not find a value for the name 'hash.int.v'
}

func ExampleLookup_badIntDig() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `hash.int.3`, nil, options)
		return nil
	}))
	// Output: lookup() did not find a value for the name 'hash.int.3'
}

func ExampleLookup2_findFirst() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup2(internal.NewInvocation(c, px.EmptyMap, nil), []string{`first`, `second`}, types.DefaultAnyType(), nil, nil, nil, options, nil))
	})
	// Output: value of first
}

func ExampleLookup2_findSecond() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup2(internal.NewInvocation(c, px.EmptyMap, nil), []string{`non existing`, `second`}, types.DefaultAnyType(), nil, nil, nil, options, nil))
	})
	// Output: includes 'value of first'
}

func ExampleLookup2_notFoundWithoutDflt() {
	printErr(hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) error {
		hiera.Lookup2(internal.NewInvocation(c, px.EmptyMap, nil), []string{`non existing`, `not there`}, types.DefaultAnyType(), nil, nil, nil, options, nil)
		return nil
	}))
	// Output: lookup() did not find a value for any of the names [non existing, not there]
}

func ExampleLookup2_notFoundDflt() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		fmt.Println(hiera.Lookup2(internal.NewInvocation(c, px.EmptyMap, nil), []string{`non existing`, `not there`}, types.DefaultAnyType(), types.WrapString(`default value`), nil, nil, options, nil))
	})
	// Output: default value
}

func ExampleLookup_dottedStringInt() {
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(c px.Context) {
		v := hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `hash.array.0`, nil, options)
		fmt.Println(v)
	})
	// Output: two
}

func ExampleLookup_mapProvider() {
	sampleData := map[string]string{
		`a`: `value of a`,
		`b`: `value of b`}

	tp := func(ic hieraapi.ServerContext, key string) px.Value {
		if v, ok := sampleData[key]; ok {
			return types.WrapString(v)
		}
		return nil
	}

	hiera.DoWithParent(context.Background(), tp, map[string]px.Value{}, func(c px.Context) {
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `a`, nil, nil))
		fmt.Println(hiera.Lookup(internal.NewInvocation(c, px.EmptyMap, nil), `b`, nil, nil))
	})

	// Output:
	// value of a
	// value of b
}
