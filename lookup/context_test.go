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

func ExampleDoWithParent() {
	sampleData := map[string]eval.PValue {
		`first`: eval.Wrap(`value of first`),
		`second`: eval.Wrap(`includes %lookup('first')'`),
	}

	provider := func(c eval.Context, key string, options eval.KeyedValue) (eval.PValue, bool, error) {
		v, ok := sampleData[key]
		return v, ok, nil
	}

	lookup.DoWithParent(context.Background(), provider, func(c eval.Context) error {
		v := c.Evaluate(c.ParseAndValidate(`sample.pp`, `lookup('first')`, false))
		fmt.Println(v)
		return nil
	})
	// Output: value of first
}
