package examples_test

import (
	"context"
	"testing"

	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/merge"
	"github.com/lyraproj/hiera/provider"
)

// TestMerge_default tests the default merge strategy which is "first found"
func TestMerge_default(t *testing.T) {
	configOptions := map[string]string{api.HieraConfig: `testdata/merge.yaml`}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		// m.a only exists in the first provider
		result := hiera.Lookup(hs.Invocation(nil, nil), `m.a`, nil, nil)
		if result == nil || `first value of a` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// m.b only exists in the second provider and is hence not found since the hashes are not merged
		result = hiera.Lookup(hs.Invocation(nil, nil), `m.b`, nil, nil)
		if result != nil {
			t.Fatalf("unexpected result %v", result)
		}

		// m.c exists in both but since no merge occurs, the first one is selected
		result = hiera.Lookup(hs.Invocation(nil, nil), `m.c`, nil, nil)
		if result == nil || `first value of c` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}
	})
}

// TestMerge_deep shows how to pass a merge option in a lookup. The possible merge options are: First,
// Unique, Hash, and Deep. Their behaviour should correspond to Puppet Hiera except for the current
// limitation that Deep cannot be fine tuned with additional options. So no "knock_out_prefix" etc. just
// yet.
//
// As with Puppet Hiera, merge options can also be specified as lookup_options in the data files.
func TestMerge_deep(t *testing.T) {
	configOptions := map[string]string{api.HieraConfig: `testdata/merge.yaml`}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(hs api.Session) {
		// options containing the merge option "deep"
		opts := map[string]string{`merge`: `deep`}

		// m.a only exists in the first provider
		result := hiera.Lookup(hs.Invocation(nil, nil), `m.a`, nil, opts)
		if result == nil || `first value of a` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// m.b only exists in the second provider
		result = hiera.Lookup(hs.Invocation(nil, nil), `m.b`, nil, opts)
		if result == nil || `second value of b` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// m.c exists in both and since a merge occurs, the first one has precedence
		result = hiera.Lookup(hs.Invocation(nil, nil), `m.c`, nil, opts)
		if result == nil || `first value of c` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// obtain the full map and compare
		result = hiera.Lookup(hs.Invocation(nil, nil), `m`, nil, opts)
		if !vf.Value(map[string]string{`a`: `first value of a`, `b`: `second value of b`, `c`: `first value of c`}).Equals(result) {
			t.Fatalf("unexpected result %v", result)
		}

		// obtain m and m2 using first found strategy (no options). Then merge them with an explicit call to DeepMerge
		m := hiera.Lookup(hs.Invocation(nil, nil), `m`, nil, nil)
		m2 := hiera.Lookup(hs.Invocation(nil, nil), `m2`, nil, nil)
		result, ok := merge.Deep(m, m2, opts)
		if !ok {
			t.Fatal("DeepMerge failed")
		}
		if !vf.Value(map[string]string{`a`: `first value of a`, `b`: `third value of b`, `c`: `first value of c`}).Equals(result) {
			t.Fatalf("unexpected result %v", result)
		}
	})
}
