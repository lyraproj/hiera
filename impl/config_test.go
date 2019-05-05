package impl_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lyraproj/pcore/types"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/hiera/impl"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
	"github.com/stretchr/testify/require"
)

func TestConfigLookup_default(t *testing.T) {
	hclog.DefaultOptions.Level = hclog.Debug
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]px.Value{impl.HieraRoot: types.WrapString(filepath.Join(wd, `testdata`, `defaultconfig`))}
	lookup.DoWithParent(context.Background(), nil, options, func(c px.Context) {
		require.Equal(t, `value of first`, lookup.Lookup(impl.NewInvocation(c, px.EmptyMap), `first`, nil, nil).String())
	})
}

func TestConfigLookup_lyra_default(t *testing.T) {
	hclog.DefaultOptions.Level = hclog.Debug
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]px.Value{impl.HieraRoot: types.WrapString(filepath.Join(wd, `testdata`, `defaultlyraconfig`))}
	lookup.DoWithParent(context.Background(), nil, options, func(c px.Context) {
		require.Equal(t, `value of first`, lookup.Lookup(impl.NewInvocation(c, px.EmptyMap), `first`, nil, nil).String())
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
		`deep`, `{'one' => 1, 'two' => 'two', 'three' => {'a' => 'A', 'c' => 'C', 'b' => 'B'}}`)
}

func TestConfigLookup_unique(t *testing.T) {
	testExplicit(t, `array`,
		`unique`, `['one', 'two', 'three', 'four', 'five']`)
}

func testExplicit(t *testing.T, key, merge, expected string) {
	t.Helper()
	hclog.DefaultOptions.Level = hclog.Debug
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]px.Value{impl.HieraRoot: types.WrapString(filepath.Join(wd, `testdata`, `explicit`))}
	luOpts := map[string]px.Value{`merge`: types.WrapString(merge)}
	lookup.DoWithParent(context.Background(), nil, options, func(c px.Context) {
		require.Equal(t, expected, lookup.Lookup(impl.NewInvocation(c, px.EmptyMap), key, nil, luOpts).String())
	})
}
