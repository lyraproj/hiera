package impl

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/hiera/lookup"
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

// HieraRoot is an option key that can be used to change the default root which is the current working directory
const HieraRoot = `Hiera::Root`

// HieraConfigFileName is an option that can be used to change the default file name 'hiera.yaml'
const HieraConfigFileName = `Hiera::ConfigFileName`

// HieraConfig is an option that can be used to change absolute path of the hiera config. When specified, the
// HieraRoot and HieraConfigFileName will not have any effect.
const HieraConfig = `Hiera::Config`

type invocation struct {
	px.Context
	nameStack  []string
	configPath string
	scope      px.Keyed

	expCtx   string
	key      lookup.Key
	provider lookup.DataProvider
	location lookup.Location
}

// InitContext initializes the given context with the Hiera cache. The context initialized
// with this method determines the life-cycle of that cache.
func InitContext(c px.Context, topProvider lookup.LookupKey, options map[string]px.Value) {
	c.Set(hieraCacheKey, &sync.Map{})
	if topProvider == nil {
		topProvider = ConfigLookup
	}
	c.Set(hieraTopProviderKey, topProvider)
	c.Set(hieraTopProviderCacheKey, make(map[string]px.Value, 23))
	c.Set(hieraGlobalOptionsKey, options)

	_, ok := options[HieraConfig]
	if !ok {
		var hieraRoot string
		r, ok := options[HieraRoot]
		if ok {
			hieraRoot = r.String()
		} else {
			var err error
			if hieraRoot, err = os.Getwd(); err != nil {
				panic(err)
			}
		}

		var fileName string
		if r, ok = options[HieraConfigFileName]; ok {
			fileName = r.String()
		} else {
			fileName = `hiera.yaml`
		}
		options[HieraConfig] = types.WrapString(filepath.Join(hieraRoot, fileName))
	}
}

func NewInvocation(c px.Context, scope px.Keyed) lookup.Invocation {
	return &invocation{Context: c, nameStack: []string{}, scope: scope, configPath: globalOptions(c)[HieraConfig].String()}
}

var first = types.WrapString(`first`)

func ConfigLookup(pc lookup.ProviderContext, key string, options map[string]px.Value) (px.Value, bool) {
	ic := pc.(*providerCtx).invocation
	cfg := ic.Config()
	merge, ok := options[`merge`]
	if ok {
		var mh px.OrderedMap
		if mh, ok = merge.(px.OrderedMap); ok {
			merge = mh.Get5(`strategy`, first)
		}
	} else {
		merge = first
	}

	k := NewKey(key)
	ms := lookup.GetMergeStrategy(merge.String())
	v := ms.Lookup(cfg.Hierarchy(), ic, func(prv interface{}) px.Value {
		pr := prv.(lookup.DataProvider)
		return pr.UncheckedLookup(k, ic, ms)
	})
	return v, v != nil
}

func (ic *invocation) topProvider() lookup.LookupKey {
	if v, ok := ic.Get(hieraTopProviderKey); ok {
		var tp lookup.LookupKey
		if tp, ok = v.(lookup.LookupKey); ok {
			return tp
		}
	}
	panic(px.Error(NotInitialized, issue.NoArgs))
}

func (ic *invocation) topProviderCache() map[string]px.Value {
	if v, ok := ic.Get(hieraTopProviderCacheKey); ok {
		var tc map[string]px.Value
		if tc, ok = v.(map[string]px.Value); ok {
			return tc
		}
	}
	panic(px.Error(NotInitialized, issue.NoArgs))
}

func globalOptions(c px.Context) map[string]px.Value {
	if v, ok := c.Get(hieraGlobalOptionsKey); ok {
		var g map[string]px.Value
		if g, ok = v.(map[string]px.Value); ok {
			return g
		}
	}
	panic(px.Error(NotInitialized, issue.NoArgs))
}

func (ic *invocation) sharedCache() *sync.Map {
	if v, ok := ic.Get(hieraCacheKey); ok {
		var sh *sync.Map
		if sh, ok = v.(*sync.Map); ok {
			return sh
		}
	}
	panic(px.Error(NotInitialized, issue.NoArgs))
}

func (ic *invocation) Config() (conf lookup.ResolvedConfig) {
	sc := ic.sharedCache()
	cp := hieraConfigsPrefix + ic.configPath
	if val, ok := sc.Load(cp); ok {
		conf = val.(lookup.ResolvedConfig)
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
		conf = val.(lookup.ResolvedConfig)
	} else {
		conf = NewConfig(ic, ic.configPath).Resolve(ic)
		sc.Store(cp, conf)
	}
	return
}

func (ic *invocation) lookupViaCache(key lookup.Key, options map[string]px.Value) (px.Value, bool) {
	rootKey := key.Root()

	sc := ic.sharedCache()
	if val, ok := sc.Load(rootKey); ok {
		return key.Dig(val.(px.Value))
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
	ic.key = key
	if v, ok := ic.topProvider()(newContext(ic, ic.topProviderCache()), rootKey, options); ok {
		v := Interpolate(ic, v, true)
		sc.Store(rootKey, v)
		return key.Dig(v)
	}
	return nil, false
}

func (ic *invocation) Check(key lookup.Key, actor px.Producer) px.Value {
	if utils.ContainsString(ic.nameStack, key.String()) {
		panic(px.Error(EndlessRecursion, issue.H{`name_stack`: ic.nameStack}))
	}
	ic.nameStack = append(ic.nameStack, key.String())
	defer func() {
		ic.nameStack = ic.nameStack[:len(ic.nameStack)-1]
	}()
	return actor()
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

func (ic *invocation) WithDataProvider(p lookup.DataProvider, actor px.Producer) px.Value {
	saveProv := ic.provider
	ic.provider = p
	defer func() {
		ic.provider = saveProv
	}()
	return actor()
}

func (ic *invocation) WithLocation(loc lookup.Location, actor px.Producer) px.Value {
	saveLoc := ic.location
	ic.location = loc
	defer func() {
		ic.location = saveLoc
	}()
	return actor()
}

func (ic *invocation) ReportLocationNotFound() {
	lg := hclog.Default()
	if lg.IsDebug() {
		lg.Debug(`location not found`, ic.debugArgs()...)
	}
}

func (ic *invocation) ReportFound(value px.Value) {
	lg := hclog.Default()
	if lg.IsDebug() {
		lg.Debug(`value found`, append(ic.debugArgs(), `key`, ic.key.String(), `value`, value.String())...)
	}
}

func (ic *invocation) ReportNotFound(key string) {
	lg := hclog.Default()
	if lg.IsDebug() {
		lg.Debug(`key not found`, append(ic.debugArgs(), `key`, key)...)
	}
}

func (ic *invocation) NotFound() {
	panic(lookup.NotFound)
}

func (ic *invocation) Explain(messageProducer func() string) {
	lg := hclog.Default()
	if lg.IsDebug() {
		lg.Debug(messageProducer())
	}
}

func (ic *invocation) WithExplanationContext(n string, f func()) {
	saveExpCtx := ic.expCtx
	defer func() {
		ic.expCtx = saveExpCtx
	}()
	ic.expCtx = n
	// TODO: Add explanation support
	f()
}

func (ic *invocation) debugArgs() []interface{} {
	args := make([]interface{}, 0, 4)
	if ic.expCtx != `` {
		args = append(args, `context`)
		args = append(args, ic.expCtx)
	}
	if ic.provider != nil {
		args = append(args, `provider`)
		args = append(args, ic.provider.FullName())
	}
	if ic.location != nil {
		args = append(args, `location`)
		args = append(args, ic.location.String())
	}
	return args
}
