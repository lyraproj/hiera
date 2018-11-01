package lookup

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
)

const HieraCacheKey = `Hiera::Cache`
const HieraConfigsKey = `Hiera::Config::`

// An Invocation keeps track of one specific lookup invocation implements a guard against
// endless recursion
type Invocation interface {
	Check(key Key, value func() eval.PValue) eval.PValue
	WithDataProvider(dh DataProvider, value func() eval.PValue) eval.PValue
	WithLocation(loc Location, value func() eval.PValue) eval.PValue
	ReportLocationNotFound()
	ReportFound(key string, value eval.PValue)
	ReportNotFound(key string)
	Context() eval.Context
}

type invocation struct {
	context eval.Context
	sharedCache *ConcurrentMap
	nameStack []string
}

// InitContext initializes the given context with the Hiera cache. The context initialized
// with this method determines the life-cycle of that cache.
func InitContext(c eval.Context) {
	c.Set(HieraCacheKey, NewConcurrentMap(37))
}

func NewInvocation(c eval.Context, configPath string) Invocation {
	if sh, ok := c.Get(HieraCacheKey); ok {
		return &invocation{context: c, sharedCache: sh.(*ConcurrentMap), nameStack: []string{}}
	}
	panic(eval.Error(c, HIERA_NOT_INITIALIZED, issue.NO_ARGS))
}

func (ic *invocation) Config(configPath string) ResolvedConfig {
	val := ic.sharedCache.EnsureSet(HieraConfigsKey + configPath, func() interface{} {
		return NewConfig(ic, configPath)
	})
	return val.(ResolvedConfig)
}

func (ic *invocation) lookupViaCache(key Key, options eval.KeyedValue) (eval.PValue, bool) {
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
		val = Interpolate(c, c.topProvider(c, rootKey, options), true)
		return
	})
	if val == notFoundSingleton {
		return nil, false
	}
	return key.Dig(c, val.(eval.PValue))
}

func (ic *invocation) Check(key Key, actor func() eval.PValue) eval.PValue {
	if utils.ContainsString(ic.nameStack, key.String()) {
		panic(eval.Error(ic.context, HIERA_ENDLESS_RECURSION, issue.H{`name_stack`: ic.nameStack}))
	}
	ic.nameStack = append(ic.nameStack, key.String())
	defer func() {
		ic.nameStack = ic.nameStack[:len(ic.nameStack)-1]
	}()
	return actor()
}

func (ic *invocation) WithDataProvider(dh DataProvider, actor func() eval.PValue) eval.PValue {
	return actor()
}

func (ic *invocation) WithLocation(loc Location, actor func() eval.PValue) eval.PValue {
	return actor()
}

func (ic *invocation) ReportLocationNotFound() {
}

func (ic *invocation) ReportFound(key string, value eval.PValue) {
}

func (ic *invocation) ReportNotFound(key string) {
}

func (ic *invocation) Context() eval.Context {
	return ic.context
}

