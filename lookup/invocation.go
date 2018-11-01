package lookup

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
	"fmt"
)

const HieraCacheKey = `Hiera::Cache`
const HieraConfigsKey = `Hiera::Config::`

// An Invocation keeps track of one specific lookup invocation implements a guard against
// endless recursion
type Invocation interface {
	Context
	Check(key Key, value func() eval.Value) eval.Value
	WithDataProvider(dh DataProvider, value func() eval.Value) eval.Value
	WithLocation(loc Location, value func() eval.Value) eval.Value
	ReportLocationNotFound()
	ReportFound(key string, value eval.Value)
	ReportNotFound(key string)
}

type invocation struct {
	lookupCtx
	sharedCache *ConcurrentMap
	nameStack []string
}

// InitContext initializes the given context with the Hiera cache. The context initialized
// with this method determines the life-cycle of that cache.
func InitContext(c eval.Context) {
	c.Set(HieraCacheKey, NewConcurrentMap(37))
}

func NewInvocation(c eval.Context) Invocation {
	lc, ok := c.(*lookupCtx)
	if !ok {
		panic(fmt.Errorf(`lookup called without lookup.Context`))
	}

	if sh, ok := c.Get(HieraCacheKey); ok {
		return &invocation{lookupCtx: *lc, sharedCache: sh.(*ConcurrentMap), nameStack: []string{}}
	}
	panic(eval.Error(HIERA_NOT_INITIALIZED, issue.NO_ARGS))
}

func (ic *invocation) Config(configPath string) ResolvedConfig {
	val := ic.sharedCache.EnsureSet(HieraConfigsKey + configPath, func() interface{} {
		return NewConfig(ic, configPath)
	})
	return val.(ResolvedConfig)
}

func (ic *invocation) lookupViaCache(key Key, options eval.OrderedMap) (eval.Value, bool) {
	rootKey := key.Root()

	val := ic.sharedCache.EnsureSet(rootKey, func() (val interface{}) {
		defer func() {
			if r := recover(); r != nil {
				if r == notFoundSingleton {
					val = r
				} else {
					panic(r)
				}
			}
		}()
		val = Interpolate(ic, ic.topProvider(ic, rootKey, options), true)
		return
	})
	if val == notFoundSingleton {
		return nil, false
	}
	return key.Dig(val.(eval.Value))
}

func (ic *invocation) Check(key Key, actor func() eval.Value) eval.Value {
	if utils.ContainsString(ic.nameStack, key.String()) {
		panic(eval.Error(HIERA_ENDLESS_RECURSION, issue.H{`name_stack`: ic.nameStack}))
	}
	ic.nameStack = append(ic.nameStack, key.String())
	defer func() {
		ic.nameStack = ic.nameStack[:len(ic.nameStack)-1]
	}()
	return actor()
}

func (ic *invocation) WithDataProvider(dh DataProvider, actor func() eval.Value) eval.Value {
	return actor()
}

func (ic *invocation) WithLocation(loc Location, actor func() eval.Value) eval.Value {
	return actor()
}

func (ic *invocation) ReportLocationNotFound() {
}

func (ic *invocation) ReportFound(key string, value eval.Value) {
}

func (ic *invocation) ReportNotFound(key string) {
}
