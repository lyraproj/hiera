package impl

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
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
	return lookup.LcPath
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

func (p *path) Original() string {
	return p.original
}

func (p *path) Resolved() string {
	return p.resolved
}

type glob struct {
	pattern string
}

func (g *glob) Exist() bool {
	return false
}

func (g *glob) Kind() lookup.LocationKind {
	return lookup.LcGlob
}

func (g *glob) String() string {
	return fmt.Sprintf("glob{pattern:%s}", g.pattern)
}

func (p *glob) Original() string {
	return p.pattern
}

func (g *glob) Resolve(ic lookup.Invocation, dataDir string) []lookup.Location {
	r, _ := interpolateString(ic, g.pattern, false)
	rp := filepath.Join(dataDir, r.String())
	matches, _ := doublestar.Glob(rp)
	locs := make([]lookup.Location, len(matches))
	for i, m := range matches {
		locs[i] = &path{g.pattern, m, true}
	}
	return locs
}

func (p *glob) Resolved() string {
	// This should never happen.
	panic(fmt.Errorf(`resolved requested on a glob`))
}

type uri struct {
	original string
	resolved string
}

func (u *uri) Exist() bool {
	return true
}

func (u *uri) Kind() lookup.LocationKind {
	return lookup.LcUri
}

func (u *uri) String() string {
	return fmt.Sprintf("uri{original:%s, resolved:%s", u.original, u.resolved)
}

func (p *uri) Original() string {
	return p.original
}

func (u *uri) Resolve(ic lookup.Invocation, dataDir string) []lookup.Location {
	r, _ := interpolateString(ic, u.original, false)
	return []lookup.Location{&uri{u.original, r.String()}}
}

func (p *uri) Resolved() string {
	return p.resolved
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
	return lookup.LcMappedPaths
}

func (m *mappedPaths) Original() string {
	return m.String()
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

func (m *mappedPaths) Resolved() string {
	// This should never happen.
	panic(fmt.Errorf(`resolved requested on mapped paths`))
}
