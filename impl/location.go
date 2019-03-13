package impl

import (
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"os"
	"path/filepath"
)

type path struct {
	original string
	resolved string
	exist    bool
}

func (p *path) Exist() bool {
	return p.exist
}

func (p *path) Kind() lookup.LocationKind {
	return lookup.LC_PATH
}

func (p *path) String() string {
	return fmt.Sprintf("path{ original:%s, resolved:%s, exist:%v}", p.original, p.resolved, p.exist)
}

func (p *path) Resolve(ic lookup.Invocation, dataDir string) []lookup.Location {
	r, _ := interpolateString(ic, p.original, false)
	rp := filepath.Join(dataDir, r.String())
	_, err := os.Stat(rp)
	return []lookup.Location{&path{p.original, rp, err == nil}}
}

type glob struct {
	pattern string
}

func (g *glob) Exist() bool {
	return false
}

func (g *glob) Kind() lookup.LocationKind {
	return lookup.LC_GLOB
}

func (g *glob) String() string {
	return fmt.Sprintf("glob{pattern:%s}", g.pattern)
}

func (g *glob) Resolve(ic lookup.Invocation, dataDir string) []lookup.Location {
	r, _ := interpolateString(ic, g.pattern, false)
	rp := filepath.Join(dataDir, r.String())
	matches, err := doublestar.Glob(rp)
	if err != nil {

	}
	locs := make([]lookup.Location, len(matches))
	for i, m := range matches {
		locs[i] = &path{g.pattern, m, true}
	}
	return locs
}

type uri struct {
	original string
	resolved string
}

func (u *uri) Exist() bool {
	return true
}

func (u *uri) Kind() lookup.LocationKind {
	return lookup.LC_URI
}

func (u *uri) String() string {
	return fmt.Sprintf("uri{original:%s, resolved:%s", u.original, u.resolved)
}

func (u *uri) Resolve(ic lookup.Invocation, dataDir string) []lookup.Location {
	r, _ := interpolateString(ic, u.original, false)
	return []lookup.Location{&uri{u.original, r.String()}}
}

type mappedPaths struct {
	// Name of variable that contains an array of strings
	sourceVar string

	// Variable name to use when resolving template
	key string

	// Template containing interpolation of the key
	template string
}

func (m *mappedPaths) Exist() bool {
	return false
}

func (m *mappedPaths) Kind() lookup.LocationKind {
	return lookup.LC_MAPPED_PATHS
}

func (m *mappedPaths) String() string {
	return fmt.Sprintf("mapped_paths{sourceVar:%s, key:%s, template:%s}", m.sourceVar, m.key, m.template)
}

func (m *mappedPaths) Resolve(ic lookup.Invocation, dataDir string) []lookup.Location {
	var mappedVars *types.Array
	v := resolveInScope(ic, m.sourceVar, false)
	if sv, ok := v.(px.StringValue); ok {
		mappedVars = types.SingletonArray(sv)
	} else {
		mva, ok := v.(*types.Array)
		if !ok || mva.Len() == 0 {
			return []lookup.Location{}
		}
		mappedVars = mva
	}

	paths := make([]lookup.Location, mappedVars.Len())

	// Use a parented scope so that the tracking scope held by the context is shielded from the
	// interpolations of the key introduced by this mapped path.
	/*
	ic.DoWithScope(impl.NewParentedScope(ic.Scope(), true), func() {
		mappedVars.EachWithIndex(func(mv px.Value, i int) {
			scope := ic.Scope()
			scope.WithLocalScope(func() px.Value {
				scope.Set(m.key, mv)
				r, _ := interpolateString(ic, m.template, false)
				rp := filepath.Join(dataDir, r.String())
				_, err := os.Stat(rp)
				paths[i] = &path{m.template, rp, err == nil}
				return nil
			})
		})
	})
	 */
	return paths
}
