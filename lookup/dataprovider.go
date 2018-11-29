package lookup

import "github.com/puppetlabs/go-evaluator/eval"

type DataProvider interface {
	UncheckedLookup(key Key, invocation Invocation, merge MergeStrategy) (eval.Value, bool)
	FullName() string
}

