package lookup

import "github.com/puppetlabs/go-evaluator/eval"

type DataDig func(c eval.Context, key Key, options eval.KeyedValue) (eval.PValue, bool, error)

type DataHash func(c eval.Context, options eval.KeyedValue) (eval.KeyedValue, error)

type LookupKey func(c eval.Context, key string, options eval.KeyedValue) (eval.PValue, bool, error)

