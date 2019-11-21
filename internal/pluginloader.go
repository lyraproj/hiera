package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/serialization"
	"github.com/lyraproj/pcore/types"
	log "github.com/sirupsen/logrus"
)

// a pluginLoader loads plugins that matches specific hierarchy entries
type pluginLoader struct {
	px.DefiningLoader
	he hieraapi.Entry
}

// a plugin corresponds to a loaded process
type plugin struct {
	lock      sync.Mutex
	wGroup    sync.WaitGroup
	process   *os.Process
	path      string
	addr      string
	functions map[string]interface{}
}

// a pluginRegistry keeps track of loaded plugins
type pluginRegistry struct {
	lock    sync.Mutex
	plugins map[string]*plugin
}

// NewPluginLoader returns a loader that is capable of discovered plugins that matches the given hierarchy entry. If
// such plugins are found, the will be added to the root loader. The loaded entry and the corresponding executable
// will be kept alive for until this executable terminates.
func NewPluginLoader(parent px.Loader, he hieraapi.Entry) px.DefiningLoader {
	return &pluginLoader{DefiningLoader: px.NewParentedLoader(parent), he: he}
}

// stopAll will stop all plugins that this registry is aware of and empty the registry
func (r *pluginRegistry) stopAll() {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, p := range r.plugins {
		p.kill()
	}
	r.plugins = nil
}

// startPlugin will start the plugin loaded from the given path and register the functions that it makes available
// with the given loader.
func (r *pluginRegistry) startPlugin(c px.Context, path string, loader px.DefiningLoader) {
	r.lock.Lock()
	defer r.lock.Unlock()

	var ok bool
	var p *plugin
	if r.plugins != nil {
		p, ok = r.plugins[path]
		if ok {
			return
		}
	}

	cmd := exec.Command(path)
	cmd.Env = []string{`HIERA_MAGIC_COOKIE=` + strconv.Itoa(hiera.MagicCookie)}

	createPipe := func(name string, fn func() (io.ReadCloser, error)) io.ReadCloser {
		pipe, err := fn()
		if err != nil {
			panic(fmt.Errorf(`unable to create %s pipe to plugin %s: %s`, name, path, err.Error()))
		}
		return pipe
	}

	cmdErr := createPipe(`stderr`, cmd.StderrPipe)
	cmdOut := createPipe(`stderr`, cmd.StdoutPipe)
	err := cmd.Start()
	if err != nil {
		panic(fmt.Errorf(`unable to start plugin %s: %s`, path, err.Error()))
	}

	// Make sure the plugin process is killed if there is an error
	defer func() {
		r := recover()
		if err != nil || r != nil {
			_ = cmd.Process.Kill()
		}
		if r != nil {
			panic(r)
		}
	}()

	p = &plugin{path: path, process: cmd.Process}

	// start a go routine that propagates everything written on the plugin's stderr to
	// the StandardLogger of this process.
	p.wGroup.Add(1)
	go func() {
		defer p.wGroup.Done()
		out := log.StandardLogger().Out
		reader := bufio.NewReaderSize(cmdErr, 0x10000)
		for {
			line, pfx, err := reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					log.Errorf(`error reading stderr of plugin %s: %s`, path, err.Error())
				}
				return
			}
			_, _ = out.Write(line)
			if !pfx {
				_, _ = out.Write([]byte{'\n'})
			}
		}
	}()

	// Start a go routine that awaits the initial meta-info from the plugin.
	metaCh := make(chan interface{})
	p.wGroup.Add(1)
	go func() {
		defer p.wGroup.Done()
		var meta map[string]interface{}
		dc := json.NewDecoder(cmdOut)
		err := dc.Decode(&meta)
		if err != nil {
			metaCh <- err
		} else {
			metaCh <- meta
		}
	}()

	// Give plugin some time to respond with meta-info
	timeout := time.After(time.Second * 3)
	var meta map[string]interface{}
	select {
	case <-timeout:
		panic(fmt.Errorf(`timeout while waiting for plugin %s to start`, path))
	case mv := <-metaCh:
		if err, ok := mv.(error); ok {
			panic(fmt.Errorf(`error reading meta data of plugin %s: %s`, path, err.Error()))
		}
		meta = mv.(map[string]interface{})
	}

	// Ignore other stuff that is written on plugin's stdout
	p.wGroup.Add(1)
	go func() {
		defer p.wGroup.Done()
		toss := make([]byte, 0x1000)
		for {
			_, err := cmdOut.Read(toss)
			if err == io.EOF {
				return
			}
		}
	}()
	if r.plugins == nil {
		r.plugins = make(map[string]*plugin)
	}
	p.initialize(meta)
	r.plugins[path] = p

	p.registerFunctions(c, loader)
}

func (p *plugin) kill() {
	p.lock.Lock()
	process := p.process
	if process == nil {
		return
	}

	defer func() {
		p.wGroup.Wait()
		p.process = nil
		p.lock.Unlock()
	}()

	// SIGINT on windows will fail
	graceful := true
	if err := process.Signal(syscall.SIGINT); err != nil {
		graceful = false
	}

	if graceful {
		done := make(chan bool)
		go func() {
			_, _ = process.Wait()
			done <- true
		}()
		select {
		case <-done:
		case <-time.After(time.Second * 3):
			_ = process.Kill()
		}
	} else {
		// Windows. Just kill it!
		_ = process.Kill()
	}
}

// initialize the plugin with the given meta-data
func (p *plugin) initialize(meta map[string]interface{}) {
	v, ok := meta[`version`].(float64)
	if !(ok && int(v) == hiera.ProtoVersion) {
		panic(fmt.Errorf(`plugin %s uses unsupported protocol %v`, p.path, v))
	}
	p.addr, ok = meta[`address`].(string)
	if !ok {
		panic(fmt.Errorf(`plugin %s did not provide a valid address`, p.path))
	}
	p.functions, ok = meta[`functions`].(map[string]interface{})
	if !ok {
		panic(fmt.Errorf(`plugin %s did not provide a valid functions map`, p.path))
	}
}

type luDispatch func(string) px.DispatchCreator

// registerFunctions will register functions found in meta-info with the given loader.
func (p *plugin) registerFunctions(c px.Context, loader px.DefiningLoader) {
	for k, v := range p.functions {
		names := v.([]interface{})
		var df luDispatch
		switch k {
		case `data_dig`:
			df = p.dataDigDispatch
		case `data_hash`:
			df = p.dataHashDispatch
		default:
			df = p.lookupKeyDispatch
		}
		for _, x := range names {
			n := x.(string)
			f := px.BuildFunction(n, nil, []px.DispatchCreator{df(n)})
			loader.SetEntry(px.NewTypedName(px.NsFunction, n), px.NewLoaderEntry(f.Resolve(c), nil))
		}
	}
}

func (p *plugin) dataDigDispatch(name string) px.DispatchCreator {
	return func(d px.Dispatch) {
		d.Param(`Hiera::Context`)
		d.Param(`Hiera::Key`)
		d.Function(func(c px.Context, args []px.Value) px.Value {
			params := makeOptions(args[0].(hieraapi.ServerContext))
			key := args[1].(hieraapi.Key)
			jp, err := json.Marshal(key.Parts())
			if err != nil {
				panic(err)
			}
			params.Add(`key`, string(jp))
			return p.callPlugin(`data_dig`, name, params)
		})
	}
}

func (p *plugin) dataHashDispatch(name string) px.DispatchCreator {
	return func(d px.Dispatch) {
		d.Param(`Hiera::Context`)
		d.Function(func(c px.Context, args []px.Value) px.Value {
			return p.callPlugin(`data_hash`, name, makeOptions(args[0].(hieraapi.ServerContext)))
		})
	}
}

func (p *plugin) lookupKeyDispatch(name string) px.DispatchCreator {
	return func(d px.Dispatch) {
		d.Param(`Hiera::Context`)
		d.Param(`String`)
		d.Function(func(c px.Context, args []px.Value) px.Value {
			params := makeOptions(args[0].(hieraapi.ServerContext))
			params.Add(`key`, args[1].String())
			return p.callPlugin(`lookup_key`, name, params)
		})
	}
}

func makeOptions(sc hieraapi.ServerContext) url.Values {
	params := make(url.Values)
	opts := make([]*types.HashEntry, 0)
	sc.EachOption(func(k string, v px.Value) {
		opts = append(opts, types.WrapHashEntry2(k, v))
	})
	if len(opts) > 0 {
		bld := bytes.Buffer{}
		serialization.DataToJson(types.WrapHash(opts), &bld)
		params.Add(`options`, strings.TrimSpace(bld.String()))
	}
	return params
}

func (p *plugin) callPlugin(luType, name string, params url.Values) px.Value {
	ad, err := url.Parse(fmt.Sprintf(`http://%s/%s/%s`, p.addr, luType, name))
	if err != nil {
		panic(err)
	}
	if len(params) > 0 {
		ad.RawQuery = params.Encode()
	}
	us := ad.String()
	client := http.Client{Timeout: time.Duration(time.Second * 5)}
	resp, err := client.Get(us)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusOK:
		vc := px.NewCollector()
		serialization.JsonToData(us, resp.Body, vc)
		return vc.Value()
	case http.StatusNotFound:
		return nil
	default:
		var bts []byte
		if bts, err = ioutil.ReadAll(resp.Body); err == nil {
			err = fmt.Errorf(`%s %s: %s`, us, resp.Status, string(bts))
		} else {
			err = fmt.Errorf(`%s %s`, us, resp.Status)
		}
		panic(err)
	}
}

func (l *pluginLoader) LoadEntry(c px.Context, name px.TypedName) px.LoaderEntry {
	entry := l.DefiningLoader.LoadEntry(c, name)
	if entry != nil {
		return entry
	}
	if name.Namespace() != px.NsFunction {
		return nil
	}

	// Get the plugin registry for this session
	var allPlugins *pluginRegistry
	if pr, ok := c.Get(hieraPluginRegistry); ok {
		allPlugins = pr.(*pluginRegistry)
	} else {
		return nil
	}

	file := l.he.PluginFile()
	if file == `` {
		file = name.Name()
		if runtime.GOOS == `windows` {
			file += `.exe`
		}
	}

	var path string
	if filepath.IsAbs(file) {
		path = filepath.Clean(file)
	} else {
		path = filepath.Clean(filepath.Join(l.he.PluginDir(), file))
		abs, err := filepath.Abs(path)
		if err != nil {
			panic(err)
		}
		path = abs
	}
	pl := l.DefiningLoader.(px.ParentedLoader).Parent()
	allPlugins.startPlugin(c, path, pl.(px.DefiningLoader))
	return pl.LoadEntry(c, name)
}

func loadPluginFunction(c px.Context, n string, he hieraapi.Entry) (fn px.Function, ok bool) {
	c.DoWithLoader(NewPluginLoader(c.Loader(), he), func() {
		var f interface{}
		if f, ok = px.Load(c, px.NewTypedName(px.NsFunction, n)); ok {
			fn = f.(px.Function)
		}
	})
	return
}
