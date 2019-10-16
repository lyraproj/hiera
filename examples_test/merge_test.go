package examples_test

import (
	"context"
	"testing"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/dgo/util"

	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/internal"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// TestMerge_default tests the default merge strategy which is "first found"
func TestMerge_default(t *testing.T) {
	configOptions := map[string]px.Value{hieraapi.HieraConfig: types.WrapString(`testdata/merge.yaml`)}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(c px.Context) {
		// m.a only exists in the first provider
		result := hiera.Lookup(internal.NewInvocation(c, nil, nil), `m.a`, nil, nil)
		if result == nil || `first value of a` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// m.b only exists in the second provider and is hence not found since the hashes are not merged
		err := util.Catch(func() {
			hiera.Lookup(internal.NewInvocation(c, nil, nil), `m.b`, nil, nil)
		})
		re, ok := err.(issue.Reported)
		if !(ok && re.Code() == hieraapi.NameNotFound) {
			t.Fatalf("unexpected error %v", err)
		}

		// m.c exists in both but since no merge occurs, the first one is selected
		result = hiera.Lookup(internal.NewInvocation(c, nil, nil), `m.c`, nil, nil)
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
	configOptions := map[string]px.Value{hieraapi.HieraConfig: types.WrapString(`testdata/merge.yaml`)}
	hiera.DoWithParent(context.Background(), provider.ConfigLookupKey, configOptions, func(c px.Context) {
		// options containing the merge option "deep"
		opts := map[string]px.Value{`merge`: px.Wrap(c, hieraapi.Deep)}

		// m.a only exists in the first provider
		result := hiera.Lookup(internal.NewInvocation(c, nil, nil), `m.a`, nil, opts)
		if result == nil || `first value of a` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// m.b only exists in the second provider
		result = hiera.Lookup(internal.NewInvocation(c, nil, nil), `m.b`, nil, opts)
		if result == nil || `second value of b` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// m.c exists in both and since a merge occurs, the first one has precedence
		result = hiera.Lookup(internal.NewInvocation(c, nil, nil), `m.c`, nil, opts)
		if result == nil || `first value of c` != result.String() {
			t.Fatalf("unexpected result %v", result)
		}

		// obtain the full map and compare
		result = hiera.Lookup(internal.NewInvocation(c, nil, nil), `m`, nil, opts)
		if !px.Wrap(c, map[string]string{`a`: `first value of a`, `b`: `second value of b`, `c`: `first value of c`}).Equals(result, nil) {
			t.Fatalf("unexpected result %v", result)
		}
	})
}
