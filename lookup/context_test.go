package lookup_test

import (
	"github.com/puppetlabs/go-hiera/lookup"
	"context"
	"github.com/puppetlabs/go-evaluator/eval"
	"fmt"

	// Ensure initialization
	_ "github.com/puppetlabs/go-evaluator/pcore"
	_ "github.com/puppetlabs/go-hiera/functions"
)

var sampleData = map[string]eval.PValue {
	`first`: eval.Wrap(`value of first`),
	`arr`: eval.Wrap([]string{`one`, `two`, `three`}),
	`hash`: eval.Wrap(map[string]interface{}{`arr`: []string{`value 1`, `value 2`}}),
	`second`: eval.Wrap(`includes %lookup('first')'`),
}

func sampleProvider(c eval.Context, key string, options eval.KeyedValue) (eval.PValue, bool, error) {
	v, ok := sampleData[key]
	return v, ok, nil
}

func ExampleDoWithParent() {
	err := lookup.DoWithParent(context.Background(), sampleProvider, func(c eval.Context) error {
		v, _ := lookup.Lookup(c, []string{`first`}, nil, nil)
		fmt.Println(v)
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// value of first
}

func ExampleLookup_dottedInt() {
	lookup.DoWithParent(context.Background(), sampleProvider, func(c eval.Context) error {
		v, _ := lookup.Lookup(c, []string{`arr.1`}, nil, nil)
		fmt.Println(v)
		return nil
	})
	// Output:
	// two
}

func ExampleLookup_dottedStringInt() {
	lookup.DoWithParent(context.Background(), sampleProvider, func(c eval.Context) error {
		v, _ := lookup.Lookup(c, []string{`hash.arr.0`}, nil, nil)
		fmt.Println(v)
		return nil
	})
	// Output:
	// value 1
}