package lookup

import (
"github.com/puppetlabs/go-evaluator/eval"
"github.com/puppetlabs/go-evaluator/types"
"path/filepath"
"fmt"
"os"
"github.com/bmatcuk/doublestar"
"github.com/puppetlabs/go-evaluator/impl"
)

type LocationKind string

const LC_PATH = LocationKind(`path`)
const LC_URI = LocationKind(`uri`)
const LC_GLOB = LocationKind(`glob`)
const LC_MAPPED_PATHS = LocationKind(`mapped_paths`)

type Location interface {
	fmt.Stringer
	Kind() LocationKind
	Exist() bool
	resolve(ic Invocation, dataDir string) []Location
}

type path struct {
	original string
	resolved string
	exist bool
}

func (p* path) Exist() bool {
	return p.exist
}

func (p* path) Kind() LocationKind {
	return LC_PATH
}

func (p* path) String() string {
	return fmt.Sprintf("path{ original:%s, resolved:%s, exist:%v}", p.original, p.resolved, p.exist)
}

func (p* path) resolve(ic Invocation, dataDir string) []Location {
	r, _ := interpolateString(ic, p.original, false)
	rp := filepath.Join(dataDir, r.String())
	_, err := os.Stat(rp)
	return []Location{&path{p.original, rp, err == nil}}
}

type glob struct {
	pattern string
}

func (g* glob) Exist() bool {
	return false
}

func (g* glob) Kind() LocationKind {
	return LC_GLOB
}

func (g* glob) String() string {
	return fmt.Sprintf("glob{pattern:%s}", g.pattern)
}

func (g* glob) resolve(ic Invocation, dataDir string) []Location {
	r, _ := interpolateString(ic, g.pattern, false)
	rp := filepath.Join(dataDir, r.String())
	matches, err := doublestar.Glob(rp)
	if err != nil {

	}
	locs := make([]Location, len(matches))
	for i, m := range matches {
		locs[i] = &path{g.pattern, m, true}
	}
	return locs
}

type uri struct {
	original string
	resolved string
}

func (u* uri) Exist() bool {
	return true
}

func (u* uri) Kind() LocationKind {
	return LC_URI
}

func (u* uri) String() string {
	return fmt.Sprintf("uri{original:%s, resolved:%s", u.original, u.resolved)
}

func (u* uri) resolve(ic Invocation, dataDir string) []Location {
	r, _ := interpolateString(ic, u.original, false)
	return []Location{&uri{u.original, r.String()}}
}

type mappedPaths struct {
	// Name of variable that contains an array of strings
	sourceVar string

	// Variable name to use when resolving template
	key string

	// Template containing interpolation of the key
	template string
}

func (m* mappedPaths) Exist() bool {
	return false
}

func (m* mappedPaths) Kind() LocationKind {
	return LC_MAPPED_PATHS
}

func (m* mappedPaths) String() string {
	return fmt.Sprintf("mapped_paths{sourceVar:%s, key:%s, template:%s}", m.sourceVar, m.key, m.template)
}

func (m* mappedPaths) resolve(ic Invocation, dataDir string) []Location {
	var mappedVars *types.ArrayValue
	v := resolveInScope(ic, m.sourceVar, false)
	if sv, ok := v.(*types.StringValue); ok {
		mappedVars = types.SingletonArray(sv)
	} else {
		mva, ok := v.(*types.ArrayValue)
		if !ok || mva.Len() == 0 {
			return []Location{}
		}
		mappedVars = mva
	}

	paths := make([]Location, mappedVars.Len())

	// Use a parented scope so that the tracking scope held by the context is shielded from the
	// interpolations of the key introduced by this mapped path.
	c := ic.Context()
	c.DoWithScope(impl.NewParentedScope(c.Scope()), func() {
		mappedVars.EachWithIndex(func(mv eval.PValue, i int) {
			scope := c.Scope()
			scope.WithLocalScope(func() eval.PValue {
				scope.Set(m.key, mv)
				r, _ := interpolateString(ic, m.template, false)
				rp := filepath.Join(dataDir, r.String())
				_, err := os.Stat(rp)
				paths[i] = &path{m.template, rp, err == nil}
				return nil
			})
		})
	})
	return paths
}
