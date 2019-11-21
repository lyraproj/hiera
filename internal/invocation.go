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

const hieraPluginRegistry = `Hiera::Plugins`

type invocation struct {
	px.Context
	nameStack  []string
	configPath string
	scope      px.Keyed
	redacted   bool
	explainer  explain.Explainer
	config     hieraapi.ResolvedConfig
}

// KillPlugins will ensure that all plugins started by this executable are gracefully terminated if possible or
// otherwise forcefully killed.
func KillPlugins(c px.Context) {
	if pr, ok := c.Get(hieraPluginRegistry); ok {
		pr.(*pluginRegistry).stopAll()
	}
}

// InitContext initializes the given context with the Hiera cache. The context initialized
// with this method determines the life-cycle of that cache.
func InitContext(c px.Context, topProvider hieraapi.LookupKey, options map[string]px.Value) {
	// Add a loader to the loader chain that will act as the parent loader for all plugin loaders
	// and be the actual cache for loaded plugins.
	c.SetLoader(px.NewParentedLoader(c.Loader()))

	c.Set(hieraCacheKey, &sync.Map{})
	if topProvider == nil {
		topProvider = provider.ConfigLookupKey
	}
	c.Set(hieraTopProviderKey, topProvider)
	c.Set(hieraTopProviderCacheKey, &sync.Map{})
	c.Set(hieraPluginRegistry, &pluginRegistry{})

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
		} else if config, ok := os.LookupEnv("HIERA_CONFIGFILE"); ok {
			fileName = config
		} else {
			fileName = `hiera.yaml`
		}
		options[hieraapi.HieraConfig] = types.WrapString(filepath.Join(hieraRoot, fileName))
	}
}

type nestedScope struct {
	parentScope px.Keyed
	scope       px.Keyed
}

func (ns *nestedScope) Get(key px.Value) (px.Value, bool) {
	v, ok := ns.scope.Get(key)
	if !ok {
		v, ok = ns.parentScope.Get(key)
	}
	return v, ok
}

func NewInvocation(c px.Context, scope px.Keyed, explainer explain.Explainer) hieraapi.Invocation {
	options := globalOptions(c)
	if gs, ok := options[hieraapi.HieraScope]; ok {
		var globalScope px.Keyed
		if globalScope, ok = gs.(px.Keyed); ok {
			if scope != nil {
				scope = &nestedScope{globalScope, scope}
			} else {
				scope = globalScope
			}
		}
	}
	if scope == nil {
		scope = px.EmptyMap
	}

	ic := &invocation{
		Context:    c,
		nameStack:  []string{},
		scope:      scope,
		configPath: options[hieraapi.HieraConfig].String(),
		explainer:  explainer}

	return ic
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

func (ic *invocation) Config() hieraapi.ResolvedConfig {
	if ic.config != nil {
		return ic.config
	}

	sc := ic.sharedCache()
	cp := hieraConfigsPrefix + ic.configPath
	if val, ok := sc.Load(cp); ok {
		ic.config = val.(hieraapi.Config).Resolve(ic)
		return ic.config
	}

	lc := hieraLockPrefix + ic.configPath

	myLock := sync.RWMutex{}
	myLock.Lock()

	var conf hieraapi.Config
	if lv, loaded := sc.LoadOrStore(lc, &myLock); loaded {
		// myLock was not stored so unlock it
		myLock.Unlock()

		if lock, ok := lv.(*sync.RWMutex); ok {
			// The loaded value is a lock. Wait for new config to be stored in place of
			// this lock
			lock.RLock()
			val, _ := sc.Load(cp)
			conf = val.(hieraapi.Config)
			lock.RUnlock()
		} else {
			conf = lv.(hieraapi.Config)
		}
	} else {
		conf = NewConfig(ic, ic.configPath)
		sc.Store(cp, conf)
		myLock.Unlock()
	}
	ic.config = conf.Resolve(ic)
	return ic.config
}

func (ic *invocation) ExplainMode() bool {
	return ic.explainer != nil
}

func (ic *invocation) lookup(key hieraapi.Key, options map[string]px.Value) px.Value {
	rootKey := key.Root()
	if rootKey == `lookup_options` {
		return ic.WithInvalidKey(key, func() px.Value {
			ic.ReportNotFound(key)
			return nil
		})
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
	v := ic.topProvider()(newServerContext(ic, ic.topProviderCache(), options), rootKey)
	if v != nil {
		dc := ic.ForData()
		v = Interpolate(dc, v, true)
		v = key.Dig(dc, v)
	}
	return v
}

func (ic *invocation) WithKey(key hieraapi.Key, actor px.Producer) px.Value {
	if utils.ContainsString(ic.nameStack, key.Source()) {
		panic(px.Error(hieraapi.EndlessRecursion, issue.H{`name_stack`: ic.nameStack}))
	}
	ic.nameStack = append(ic.nameStack, key.Source())
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
