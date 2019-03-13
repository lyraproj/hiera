package lookup

import (
	"github.com/lyraproj/pcore/px"
)

type MergeStrategy interface {
	Lookup(locations []Location, invocation Invocation, value func(location Location) (px.Value, bool)) (px.Value, bool)
}
