package hieraapi

import "fmt"

type LocationKind string

const LcPath = LocationKind(`path`)
const LcUri = LocationKind(`uri`)
const LcGlob = LocationKind(`glob`)
const LcMappedPaths = LocationKind(`mapped_paths`)

type Location interface {
	fmt.Stringer
	Kind() LocationKind
	Exist() bool
	Resolve(ic Invocation, dataDir string) []Location
	Original() string
	Resolved() string
}
