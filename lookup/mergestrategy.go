package lookup

import (
	"github.com/lyraproj/puppet-evaluator/eval"
)

type MergeStrategy interface {
	Lookup(locations []Location, invocation Invocation, value func(location Location) (eval.Value, bool)) (eval.Value, bool)
}
