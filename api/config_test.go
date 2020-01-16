package api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	"github.com/stretchr/testify/require"
)

func TestConfigLookup_default(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]string{api.HieraRoot: filepath.Join(wd, `testdata`, `defaultconfig`)}
	hiera.DoWithParent(context.Background(), nil, options, func(hs api.Session) {
		require.Equal(t, `value of first`, hiera.Lookup(hs.Invocation(nil, nil), `first`, nil, nil).String())
	})
}

func TestConfigLookup_lyra_default(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]string{api.HieraRoot: filepath.Join(wd, `testdata`, `defaultlyraconfig`)}
	hiera.DoWithParent(context.Background(), nil, options, func(hs api.Session) {
		require.Equal(t, `value of first`, hiera.Lookup(hs.Invocation(nil, nil), `first`, nil, nil).String())
	})
}

func TestConfigLookup_explicit(t *testing.T) {
	testExplicit(t, `first`, `first`, `value of first`)
}

func TestConfigLookup_hash_merge(t *testing.T) {
	testExplicit(t, `hash`,
		`hash`, `{"one":1,"three":{"a":"A","c":"C"},"two":"two"}`)
}

func TestConfigLookup_deep_merge(t *testing.T) {
	testExplicit(t, `hash`,
		``, `{"one":1,"two":"two","three":{"a":"A","c":"C","b":"B"}}`)
}

func TestConfigLookup_unique(t *testing.T) {
	testExplicit(t, `array`,
		`unique`, `{"one","two","three","four","five"}`)
}

func TestConfigLookup_sensitive(t *testing.T) {
	testExplicit(t, `sense`,
		``, `sensitive [value redacted]`)
}

func testExplicit(t *testing.T, key, merge, expected string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	options := map[string]string{api.HieraRoot: filepath.Join(wd, `testdata`, `explicit`)}
	var luOpts map[string]string
	if merge != `` {
		luOpts = map[string]string{`merge`: merge}
	}
	hiera.DoWithParent(context.Background(), nil, options, func(hs api.Session) {
		require.Equal(t, expected, hiera.Lookup(hs.Invocation(nil, nil), key, nil, luOpts).String())
	})
}
