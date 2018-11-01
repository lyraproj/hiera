package lookup

import "github.com/puppetlabs/go-evaluator/eval"

type MergeStrategy interface {
	Lookup(locations []Location, invocation Invocation, value func(location Location) eval.PValue) eval.PValue
}
