package lookup

import "github.com/puppetlabs/go-evaluator/eval"

type DataDig func(ic Invocation, key Key, options eval.OrderedMap) eval.Value

type DataHash func(ic Invocation, options eval.OrderedMap) eval.OrderedMap

type LookupKey func(ic Invocation, key string, options eval.OrderedMap) eval.Value
