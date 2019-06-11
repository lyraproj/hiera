package internal

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/lyraproj/hiera/explain"

	"github.com/lyraproj/hiera/hieraapi"

	"github.com/lyraproj/hiera/provider"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
)

const hieraCacheKey = `Hiera::Cache`
const hieraTopProviderKey = `Hiera::TopProvider`
const hieraGlobalOptionsKey = `Hiera::GlobalOptions`
const hieraTopProviderCacheKey = `Hiera::TopProvider::Cache`

const hieraConfigsPrefix = `HieraConfig:`
const hieraLockPrefix = `HieraLock:`

type invocation struct {
	px.Context
	nameStack  []string
	configPath string
	scope      px.Keyed
	redacted   bool
	explainer  explain.Explainer
}

// InitContext initializes the given context with the Hiera cache. The context initialized
// with this method determines the life-cycle of that cache.
func InitContext(c px.Context, topProvider hieraapi.LookupKey, options map[string]px.Value) {
	c.Set(hieraCacheKey, &sync.Map{})
	if topProvider == nil {
		topProvider = provider.ConfigLookupKey
	}
	c.Set(hieraTopProviderKey, topProvider)
	c.Set(hieraTopProviderCacheKey, &sync.Map{})

	if options == nil {
		options = make(map[string]px.Value)
	}
	c.Set(hieraGlobalOptionsKey, options)

	_, ok := options[hieraapi.HieraConfig]
	if !ok {
		var hieraRoot string
		r, ok := options[hieraapi.HieraRoot]
		if ok {
			hieraRoot = r.String()
		} else {
			var err error
			if hieraRoot, err = os.Getwd(); err != nil {
				panic(err)
			}
		}

		var fileName string
		if r, ok = options[hieraapi.HieraConfigFileName]; ok {
			fileName = r.String()
		} else {
			fileName = `hiera.yaml`
		}
		options[hieraapi.HieraConfig] = types.WrapString(filepath.Join(hieraRoot, fileName))
	}
}

func NewInvocation(c px.Context, scope px.Keyed, explainer explain.Explainer) hieraapi.Invocation {
	return &invocation{
		Context:    c,
		nameStack:  []string{},
		scope:      scope,
		configPath: globalOptions(c)[hieraapi.HieraConfig].String(),
		explainer:  explainer}
}

func (ic *invocation) topProvider() hieraapi.LookupKey {
	if v, ok := ic.Get(hieraTopProviderKey); ok {
		var tp hieraapi.LookupKey
		if tp, ok = v.(hieraapi.LookupKey); ok {
			return tp
		}
	}
	panic(px.Error(hieraapi.NotInitialized, issue.NoArgs))
}

func (ic *invocation) topProviderCache() *sync.Map {
	if v, ok := ic.Get(hieraTopProviderCacheKey); ok {
		var tc *sync.Map
		if tc, ok = v.(*sync.Map); ok {
			return tc
		}
	}
	panic(px.Error(hieraapi.NotInitialized, issue.NoArgs))
}

func globalOptions(c px.Context) map[string]px.Value {
	if v, ok := c.Get(hieraGlobalOptionsKey); ok {
		var g map[string]px.Value
		if g, ok = v.(map[string]px.Value); ok {
			return g
		}
	}
	panic(px.Error(hieraapi.NotInitialized, issue.NoArgs))
}

func (ic *invocation) sharedCache() *sync.Map {
	if v, ok := ic.Get(hieraCacheKey); ok {
		var sh *sync.Map
		if sh, ok = v.(*sync.Map); ok {
			return sh
		}
	}
	panic(px.Error(hieraapi.NotInitialized, issue.NoArgs))
}

func (ic *invocation) Config() (conf hieraapi.ResolvedConfig) {
	sc := ic.sharedCache()
	cp := hieraConfigsPrefix + ic.configPath
	if val, ok := sc.Load(cp); ok {
		conf = val.(hieraapi.ResolvedConfig)
		return
	}

	lc := hieraLockPrefix + ic.configPath
	myLock := sync.RWMutex{}
	myLock.Lock()
	defer myLock.Unlock()

	if lv, loaded := sc.LoadOrStore(lc, &myLock); loaded {
		// Only the one storing thread should proceed and create the configuration. This thread
		// awaits the completion of that creation by waiting for the loaded mutex.
		lock := lv.(*sync.RWMutex)
		lock.RLock()
		val, _ := sc.Load(cp)
		lock.RUnlock()
		conf = val.(hieraapi.ResolvedConfig)
	} else {
		conf = NewConfig(ic, ic.configPath).Resolve(ic)
		sc.Store(cp, conf)
	}
	return
}

func (ic *invocation) ExplainMode() bool {
	return ic.explainer != nil
}

func (ic *invocation) lookupViaCache(key hieraapi.Key, options map[string]px.Value) px.Value {
	rootKey := key.Root()
	if rootKey == `lookup_options` {
		return ic.WithInvalidKey(key, func() px.Value {
			ic.ReportNotFound(key)
			return nil
		})
	}

	sc := ic.sharedCache()
	if val, ok := sc.Load(rootKey); ok {
		return key.Dig(ic.ForData(), val.(px.Value))
	}

	globalOptions := globalOptions(ic)
	if len(options) == 0 {
		options = globalOptions
	} else if len(globalOptions) > 0 {
		no := make(map[string]px.Value, len(options)+len(globalOptions))
		for k, v := range globalOptions {
			no[k] = v
		}
		for k, v := range options {
			no[k] = v
		}
		options = no
	}
	v := ic.topProvider()(newProviderContext(ic, ic.topProviderCache()), rootKey, options)
	if v != nil {
		dc := ic.ForData()
		v = Interpolate(dc, v, true)
		sc.Store(rootKey, v)
		v = key.Dig(dc, v)
	}
	return v
}

func (ic *invocation) WithKey(key hieraapi.Key, actor px.Producer) px.Value {
	if utils.ContainsString(ic.nameStack, key.String()) {
		panic(px.Error(hieraapi.EndlessRecursion, issue.H{`name_stack`: ic.nameStack}))
	}
	ic.nameStack = append(ic.nameStack, key.String())
	defer func() {
		ic.nameStack = ic.nameStack[:len(ic.nameStack)-1]
	}()
	return actor()
}

func (ic *invocation) DoRedacted(doer px.Doer) {
	if ic.redacted {
		doer()
	} else {
		defer func() {
			ic.redacted = false
		}()
		ic.redacted = true
		doer()
	}
}

func (ic *invocation) DoWithScope(scope px.Keyed, doer px.Doer) {
	sc := ic.scope
	ic.scope = scope
	doer()
	ic.scope = sc
}

func (ic *invocation) Scope() px.Keyed {
	return ic.scope
}

func (ic *invocation) WithDataProvider(p hieraapi.DataProvider, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushDataProvider(p)
	return actor()
}

func (ic *invocation) WithInterpolation(expr string, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushInterpolation(expr)
	return actor()
}

func (ic *invocation) WithInvalidKey(key interface{}, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushInvalidKey(key)
	return actor()
}

func (ic *invocation) WithLocation(loc hieraapi.Location, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushLocation(loc)
	return actor()
}

func (ic *invocation) WithLookup(key hieraapi.Key, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushLookup(key)
	return actor()
}

func (ic *invocation) WithMerge(ms hieraapi.MergeStrategy, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushMerge(ms)
	return actor()
}

func (ic *invocation) WithSegment(seg interface{}, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushSegment(seg)
	return actor()
}

func (ic *invocation) WithSubLookup(key hieraapi.Key, actor px.Producer) px.Value {
	if ic.explainer == nil {
		return actor()
	}
	defer ic.explainer.Pop()
	ic.explainer.PushSubLookup(key)
	return actor()
}

func (ic *invocation) ForConfig() hieraapi.Invocation {
	if ic.explainer == nil {
		return ic
	}
	lic := *ic
	lic.explainer = nil
	return &lic
}

func (ic *invocation) ForData() hieraapi.Invocation {
	if ic.explainer == nil || !ic.explainer.OnlyOptions() {
		return ic
	}
	lic := *ic
	lic.explainer = nil
	return &lic
}

func (ic *invocation) ForLookupOptions() hieraapi.Invocation {
	if ic.explainer == nil || ic.explainer.Options() || ic.explainer.OnlyOptions() {
		return ic
	}
	lic := *ic
	lic.explainer = nil
	return &lic
}

func (ic *invocation) ReportLocationNotFound() {
	if ic.explainer != nil {
		ic.explainer.AcceptLocationNotFound()
	}
}

func (ic *invocation) ReportFound(key interface{}, value px.Value) {
	if ic.explainer != nil {
		ic.explainer.AcceptFound(key, value)
	}
}

func (ic *invocation) ReportMergeResult(value px.Value) {
	if ic.explainer != nil {
		ic.explainer.AcceptMergeResult(value)
	}
}

func (ic *invocation) ReportMergeSource(source string) {
	if ic.explainer != nil {
		ic.explainer.AcceptMergeSource(source)
	}
}

func (ic *invocation) ReportNotFound(key interface{}) {
	if ic.explainer != nil {
		ic.explainer.AcceptNotFound(key)
	}
}

func (ic *invocation) ReportText(messageProducer func() string) {
	if ic.explainer != nil {
		ic.explainer.AcceptText(messageProducer())
	}
}

func (ic *invocation) NotFound() {
	panic(hieraapi.NotFound)
}
