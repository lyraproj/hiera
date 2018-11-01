package lookup

import "github.com/puppetlabs/go-evaluator/eval"

type DataDig func(c Context, key Key, options eval.OrderedMap) eval.Value

type DataHash func(c Context, options eval.OrderedMap) eval.OrderedMap

type LookupKey func(c Context, key string, options eval.OrderedMap) eval.Value
