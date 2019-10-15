package hieraapi

import "fmt"

type LocationKind string

const LcPath = LocationKind(`path`)
const LcURI = LocationKind(`uri`)
const LcGlob = LocationKind(`glob`)
const LcMappedPaths = LocationKind(`mapped_paths`)

type Location interface {
	fmt.Stringer
	Kind() LocationKind
	Exists() bool
	Resolve(ic Invocation, dataDir string) []Location
	Original() string
	Resolved() string
}
