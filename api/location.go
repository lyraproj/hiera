package api

import (
	"github.com/lyraproj/dgo/dgo"
)

// LocationKind describes the kind of location that is used in a Hiera entry.
type LocationKind string

// LcPath indicates that the location is a path in a file system
const LcPath = LocationKind(`path`)

// LcURI indicates that the location is URI
const LcURI = LocationKind(`uri`)

// LcGlob indicates that the location is glob
const LcGlob = LocationKind(`glob`)

// LcMappedPaths indicates that the location is thee element array that describes a mapped path
const LcMappedPaths = LocationKind(`mapped_paths`)

// Location represents a location in a hierarchy entry and can be in the form path, uri, glob, and mapped paths.
type Location interface {
	dgo.Value
	Kind() LocationKind
	Exists() bool
	Resolve(ic Invocation, dataDir string) []Location
	Original() string
	Resolved() string
}
