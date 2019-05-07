package internal

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/lyraproj/hiera/hieraapi"

	"github.com/lyraproj/hiera/provider"

	"github.com/hashicorp/go-hclog"
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

	expCtx   string
	provider hieraapi.DataProvider
	location hieraapi.Location
	redacted bool
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

func NewInvocation(c px.Context, scope px.Keyed) hieraapi.Invocation {
	return &invocation{Context: c, nameStack: []string{}, scope: scope, configPath: globalOptions(c)[hieraapi.HieraConfig].String()}
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

func (ic *invocation) lookupViaCache(key hieraapi.Key, options map[string]px.Value) px.Value {
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
	v := ic.topProvider()(newProviderContext(ic, ic.topProviderCache()), rootKey, options)
	if v != nil {
		v := Interpolate(ic, v, true)
		sc.Store(rootKey, v)
		return key.Dig(v)
	}
	return nil
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
	saveProv := ic.provider
	ic.provider = p
	defer func() {
		ic.provider = saveProv
	}()
	return actor()
}

func (ic *invocation) WithLocation(loc hieraapi.Location, actor px.Producer) px.Value {
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
		var vs string
		if ic.redacted {
			// Value hasn't been assembled yet so it's not yet converted to a Sensitive
			vs = `value redacted`
		} else {
			vs = value.String()
		}
		lg.Debug(`value found`, append(ic.debugArgs(), `value`, vs)...)
	}
}

func (ic *invocation) ReportNotFound() {
	lg := hclog.Default()
	if lg.IsDebug() {
		lg.Debug(`key not found`, append(ic.debugArgs())...)
	}
}

func (ic *invocation) NotFound() {
	panic(hieraapi.NotFound)
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
	if len(ic.nameStack) > 0 {
		args = append(args, `key`)
		args = append(args, ic.nameStack[len(ic.nameStack)-1])
	}
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
