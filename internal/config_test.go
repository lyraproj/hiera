package internal_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/stretchr/testify/require"
)

func TestConfigLookup_default(t *testing.T) {
	hclog.DefaultOptions.Level = hclog.Debug
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]px.Value{hieraapi.HieraRoot: types.WrapString(filepath.Join(wd, `testdata`, `defaultconfig`))}
	hiera.DoWithParent(context.Background(), nil, options, func(c px.Context) {
		require.Equal(t, `value of first`, hiera.Lookup(hiera.NewInvocation(c, px.EmptyMap, nil), `first`, nil, nil).String())
	})
}

func TestConfigLookup_lyra_default(t *testing.T) {
	hclog.DefaultOptions.Level = hclog.Debug
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]px.Value{hieraapi.HieraRoot: types.WrapString(filepath.Join(wd, `testdata`, `defaultlyraconfig`))}
	hiera.DoWithParent(context.Background(), nil, options, func(c px.Context) {
		require.Equal(t, `value of first`, hiera.Lookup(hiera.NewInvocation(c, px.EmptyMap, nil), `first`, nil, nil).String())
	})
}

func TestConfigLookup_explicit(t *testing.T) {
	testExplicit(t, `first`, `first`, `value of first`)
}

func TestConfigLookup_hash_merge(t *testing.T) {
	testExplicit(t, `hash`,
		`hash`, `{'one' => 1, 'three' => {'a' => 'A', 'c' => 'C'}, 'two' => 'two'}`)
}

func TestConfigLookup_deep_merge(t *testing.T) {
	testExplicit(t, `hash`,
		``, `{'one' => 1, 'two' => 'two', 'three' => {'a' => 'A', 'c' => 'C', 'b' => 'B'}}`)
}

func TestConfigLookup_unique(t *testing.T) {
	testExplicit(t, `array`,
		`unique`, `['one', 'two', 'three', 'four', 'five']`)
}

func TestConfigLookup_sensitive(t *testing.T) {
	testExplicit(t, `sense`,
		``, `Sensitive [value redacted]`)
}

func testExplicit(t *testing.T, key, merge, expected string) {
	t.Helper()
	hclog.DefaultOptions.Level = hclog.Debug
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]px.Value{hieraapi.HieraRoot: types.WrapString(filepath.Join(wd, `testdata`, `explicit`))}
	var luOpts map[string]px.Value
	if merge != `` {
		luOpts = map[string]px.Value{`merge`: types.WrapString(merge)}
	}
	hiera.DoWithParent(context.Background(), nil, options, func(c px.Context) {
		require.Equal(t, expected, hiera.Lookup(hiera.NewInvocation(c, px.EmptyMap, nil), key, nil, luOpts).String())
	})
}
