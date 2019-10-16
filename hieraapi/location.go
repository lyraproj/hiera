package hieraapi

import (
	"github.com/lyraproj/dgo/dgo"
)

type LocationKind string

const LcPath = LocationKind(`path`)
const LcURI = LocationKind(`uri`)
const LcGlob = LocationKind(`glob`)
const LcMappedPaths = LocationKind(`mapped_paths`)

type Location interface {
	dgo.Value
	Kind() LocationKind
	Exists() bool
	Resolve(ic Invocation, dataDir string) []Location
	Original() string
	Resolved() string
}
