package hiera_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lyraproj/dgo/dgo"
	require "github.com/lyraproj/dgo/dgo_test"
	"github.com/lyraproj/dgo/typ"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/provider"
	sdk "github.com/lyraproj/hierasdk/hiera"
)

var options = vf.Map(`path`, `./testdata/sample_data.yaml`)

func TestLookup_first(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `value of first`, hiera.Lookup(iv, `first`, nil, nil))
	})
}

func TestLookup_dottedInt(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `two`, hiera.Lookup(iv, `array.1`, nil, nil).String())
	})
}

func TestLookup_dottedMix(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `value of first`,
			hiera.Lookup(iv, `hash.array.1`, nil, nil).String())
	})
}

func TestLookup_interpolate(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `includes 'value of first'`,
			hiera.Lookup(iv, `second`, nil, nil).String())
	})
}

func TestLookup_interpolateScope(t *testing.T) {
	s := map[string]string{
		`world`: `cruel world`,
	}
	testLookup(t, func(hs api.Session) {
		require.Equal(t, `hello cruel world`, hiera.Lookup(hs.Invocation(s, nil), `ipScope`, nil, nil))
		require.Equal(t, `hello cruel world`, hiera.Lookup(hs.Invocation(s, nil), `ipScope2`, nil, nil))
	})
}

func TestLookup_interpolateEmpty(t *testing.T) {
	testLookup(t, func(hs api.Session) {
		require.Equal(t, `StartEnd`, hiera.Lookup(hs.Invocation(nil, nil), `empty1`, nil, nil))
		require.Equal(t, `StartEnd`, hiera.Lookup(hs.Invocation(nil, nil), `empty2`, nil, nil))
		require.Equal(t, `StartEnd`, hiera.Lookup(hs.Invocation(nil, nil), `empty3`, nil, nil))
		require.Equal(t, `StartEnd`, hiera.Lookup(hs.Invocation(nil, nil), `empty4`, nil, nil))
		require.Equal(t, `StartEnd`, hiera.Lookup(hs.Invocation(nil, nil), `empty5`, nil, nil))
		require.Equal(t, `StartEnd`, hiera.Lookup(hs.Invocation(nil, nil), `empty6`, nil, nil))
	})
}

func TestLookup_interpolateLiteral(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `some literal text`, hiera.Lookup(iv, `ipLiteral`, nil, options))
	})
}

func TestLookup_interpolateAlias(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, vf.Strings("one", "two", "three"), hiera.Lookup(iv, `ipAlias`, nil, options))
	})
}

func TestLookup_interpolateBadAlias(t *testing.T) {
	require.NotOk(t, `'alias'/'strict_alias' interpolation is only permitted if the expression is equal to the entire string`,
		hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(hs api.Session) error {
			hiera.Lookup(hs.Invocation(nil, nil), `ipBadAlias`, nil, options)
			return nil
		}))
}

func TestLookup_interpolateBadFunction(t *testing.T) {
	require.NotOk(t, `unknown interpolation method 'bad'`,
		hiera.TryWithParent(context.Background(), provider.YamlLookupKey, options, func(hs api.Session) error {
			hiera.Lookup(hs.Invocation(nil, nil), `ipBad`, nil, options)
			return nil
		}))
}

func TestLookup_notFoundWithoutDefault(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Nil(t, hiera.Lookup(iv, `nonexistent`, nil, options))
	})
}

func TestLookup_notFoundDflt(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `default value`, hiera.Lookup(iv, `nonexistent`, vf.String(`default value`), options))
	})
}

func TestLookup_notFoundDottedIdx(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `default value`, hiera.Lookup(iv, `array.3`, vf.String(`default value`), options))
	})
}

func TestLookup_notFoundDottedMix(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `default value`, hiera.Lookup(iv, `hash.float`, vf.String(`default value`), options))
	})
}

func TestLookup_badStringDig(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Nil(t, hiera.Lookup(iv, `hash.int.v`, nil, options))
	})
}

func TestLookup_badIntDig(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Nil(t, hiera.Lookup(iv, `hash.int.3`, nil, options))
	})
}

func TestLookup2_findFirst(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `value of first`, hiera.Lookup2(iv, []string{`first`, `second`}, typ.Any, nil, nil, nil, options, nil))
	})
}

func TestLookup2_findSecond(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `includes 'value of first'`, hiera.Lookup2(iv, []string{`non existing`, `second`}, typ.Any, nil, nil, nil, options, nil))
	})
}

func TestLookup2_notFoundWithoutDflt(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Nil(t, hiera.Lookup2(iv, []string{`non existing`, `not there`}, typ.Any, nil, nil, nil, options, nil))
	})
}

func TestLookup2_notFoundDflt(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `default value`, hiera.Lookup2(iv, []string{`non existing`, `not there`}, typ.Any, vf.String(`default value`), nil, nil, options, nil))
	})
}

func TestLookup_dottedStringInt(t *testing.T) {
	testOneLookup(t, func(iv api.Invocation) {
		require.Equal(t, `two`, hiera.Lookup(iv, `hash.array.0`, nil, options))
	})
}

func ExampleLookup_mapProvider() {
	sampleData := map[string]string{
		`a`: `value of a`,
		`b`: `value of b`}

	tp := func(ic sdk.ProviderContext, key string) dgo.Value {
		if v, ok := sampleData[key]; ok {
			return vf.String(v)
		}
		return nil
	}

	hiera.DoWithParent(context.Background(), tp, nil, func(hs api.Session) {
		fmt.Println(hiera.Lookup(hs.Invocation(nil, nil), `a`, nil, nil))
		fmt.Println(hiera.Lookup(hs.Invocation(nil, nil), `b`, nil, nil))
	})

	// Output:
	// value of a
	// value of b
}

func testOneLookup(t *testing.T, f func(i api.Invocation)) {
	t.Helper()
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(hs api.Session) {
		t.Helper()
		f(hs.Invocation(nil, nil))
	})
}

func testLookup(t *testing.T, f func(hs api.Session)) {
	t.Helper()
	hiera.DoWithParent(context.Background(), provider.YamlLookupKey, options, func(hs api.Session) {
		t.Helper()
		f(hs)
	})
}
