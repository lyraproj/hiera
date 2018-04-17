package lookup

import "github.com/puppetlabs/go-evaluator/eval"

type DataDig func(c Context, key Key, options eval.KeyedValue) eval.PValue

type DataHash func(c Context, options eval.KeyedValue) eval.KeyedValue

type LookupKey func(c Context, key string, options eval.KeyedValue) eval.PValue
