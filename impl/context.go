package impl

import (
	"context"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-hiera/lookup"
	"github.com/puppetlabs/go-issues/issue"
)

func init() {
	lookup.TryWithParent = func(parent context.Context, tp lookup.LookupKey, consumer func(eval.Context) error) error {
		return eval.Puppet.TryWithParent(parent, func(c eval.Context) error {
			InitContext(c, tp)
			return consumer(c)
		})
	}

	lookup.DoWithParent = func(parent context.Context, tp lookup.LookupKey, consumer func(eval.Context)) {
		eval.Puppet.DoWithParent(parent, func(c eval.Context) {
			InitContext(c, tp)
			consumer(c)
		})
	}

	lookup.Lookup2 = func(
			ic lookup.Invocation,
			names []string,
			valueType eval.Type,
			defaultValue eval.Value,
			override eval.OrderedMap,
			defaultValuesHash eval.OrderedMap,
			options eval.OrderedMap,
			block eval.Lambda) eval.Value {
		for _, name := range names {
			key := NewKey(name)
			if v, ok := ic.Check(key, func() (eval.Value, bool) {
				return ic.(*invocation).lookupViaCache(key, options)
			}); ok {
				return v
			}
		}
		if defaultValue == nil {
			// nil (as opposed to UNDEF) means that no default was provided.
			if len(names) == 1 {
				panic(eval.Error(HIERA_NAME_NOT_FOUND, issue.H{`name`: names[0]}))
			}
			panic(eval.Error(HIERA_NOT_ANY_NAME_FOUND, issue.H{`name_list`: names}))
		}
		return defaultValue
	}
}
