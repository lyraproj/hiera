package lookup

import "fmt"

type LocationKind string

const LC_PATH = LocationKind(`path`)
const LC_URI = LocationKind(`uri`)
const LC_GLOB = LocationKind(`glob`)
const LC_MAPPED_PATHS = LocationKind(`mapped_paths`)

type Location interface {
	fmt.Stringer
	Kind() LocationKind
	Exist() bool
	Resolve(ic Invocation, dataDir string) []Location
}
