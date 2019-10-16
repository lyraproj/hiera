package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/loader"
	"github.com/lyraproj/dgo/streamer"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hierasdk/hiera"
	"github.com/lyraproj/pcore/px"
	log "github.com/sirupsen/logrus"
)

// a plugin corresponds to a loaded process
type plugin struct {
	lock      sync.Mutex
	wGroup    sync.WaitGroup
	process   *os.Process
	path      string
	addr      string
	network   string
	functions map[string]interface{}
}

// a pluginRegistry keeps track of loaded plugins
type pluginRegistry struct {
	lock    sync.Mutex
	plugins map[string]*plugin
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

var DefaultUnixSocketDir = "/tmp"

// getUnixSocketDir resolves value of unixSocketDir
func getUnixSocketDir(c px.Context) string {
	v := extractOptFromContext(c, "unixSocketDir")

	if v == "" {
		return DefaultUnixSocketDir
	}

	return v
}

var DefaultPluginTransport = "unix"

// getPluginTransport resolves value of pluginTransport
func getPluginTransport(c px.Context) string {
	v := extractOptFromContext(c, "pluginTransport")

	switch v {
	case
		"unix",
		"tcp":
		return v
	}

	return DefaultPluginTransport
}

// startPlugin will start the plugin loaded from the given path and register the functions that it makes available
// with the given loader.
func (r *pluginRegistry) startPlugin(path string) dgo.Value {
	r.lock.Lock()
	defer r.lock.Unlock()

	var ok bool
	var p *plugin
	if r.plugins != nil {
		p, ok = r.plugins[path]
		if ok {
			return p.functionMap()
		}
	}

	cmd := exec.Command(path)
	cmd.Env = []string{`HIERA_MAGIC_COOKIE=` + strconv.Itoa(hiera.MagicCookie)}
	cmd.Env = append(cmd.Env, `HIERA_PLUGIN_SOCKET_DIR=`+getUnixSocketDir(c))
	cmd.Env = append(cmd.Env, `HIERA_PLUGIN_TRANSPORT=`+getPluginTransport(c))

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

	return p.functionMap()
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
	p.network, ok = meta[`network`].(string)
	if !ok {
		log.Printf(`plugin %s did not provide a valid network, assuming tcp`, p.path)
		p.network = `tcp`
	}
	p.functions, ok = meta[`functions`].(map[string]interface{})
	if !ok {
		panic(fmt.Errorf(`plugin %s did not provide a valid functions map`, p.path))
	}
}

type luDispatch func(string) dgo.Function

func (p *plugin) functionMap() dgo.Value {
	m := vf.MutableMap()
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
			m.Put(n, df(n))
		}
	}
	return loader.Multiple(m)
}

func (p *plugin) dataDigDispatch(name string) dgo.Function {
	return vf.Value(func(pc hiera.ProviderContext, key dgo.Array) dgo.Value {
		params := makeOptions(pc)
		jp := streamer.MarshalJSON(key, nil)
		params.Add(`key`, string(jp))
		return p.callPlugin(`data_dig`, name, params)
	}).(dgo.Function)
}

func (p *plugin) dataHashDispatch(name string) dgo.Function {
	return vf.Value(func(pc hiera.ProviderContext) dgo.Value {
		return p.callPlugin(`data_hash`, name, makeOptions(pc))
	}).(dgo.Function)
}

func (p *plugin) lookupKeyDispatch(name string) dgo.Function {
	return vf.Value(func(pc hiera.ProviderContext, key string) dgo.Value {
		params := makeOptions(pc)
		params.Add(`key`, key)
		return p.callPlugin(`lookup_key`, name, params)
	}).(dgo.Function)
}

func makeOptions(pc hiera.ProviderContext) url.Values {
	params := make(url.Values)
	opts := pc.OptionsMap()
	if opts.Len() > 0 {
		bld := bytes.Buffer{}
		streamer.New(nil, streamer.DefaultOptions()).Stream(opts, streamer.JSON(&bld))
		params.Add(`options`, strings.TrimSpace(bld.String()))
	}
	return params
}

func (p *plugin) callPlugin(luType, name string, params url.Values) px.Value {
	var ad *url.URL
	var err error

	if p.network == "unix" {
		ad, err = url.Parse(fmt.Sprintf(`http://%s/%s/%s`, p.network, luType, name))
	} else {
		ad, err = url.Parse(fmt.Sprintf(`http://%s/%s/%s`, p.addr, luType, name))
	}
	if err != nil {
		panic(err)
	}
	if len(params) > 0 {
		ad.RawQuery = params.Encode()
	}
	us := ad.String()
	client := http.Client{
		Timeout: time.Duration(time.Second * 5),
		Transport: &http.Transport{
			Dial: func(_, _ string) (net.Conn, error) {
				return net.Dial(p.network, p.addr)
			},
		},
	}
	resp, err := client.Get(us)
	if err != nil {
		panic(err.Error())
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusOK:
		var bts []byte
		if bts, err = ioutil.ReadAll(resp.Body); err == nil {
			return streamer.UnmarshalJSON(bts, nil)
		}
	case http.StatusNotFound:
		return nil
	default:
		var bts []byte
		if bts, err = ioutil.ReadAll(resp.Body); err == nil {
			err = fmt.Errorf(`%s %s: %s`, us, resp.Status, string(bts))
		} else {
			err = fmt.Errorf(`%s %s`, us, resp.Status)
		}
	}
	panic(err)
}

func extractOptFromContext(c px.Context, key string) string {
	pl, ok := c.DefiningLoader().(*pluginLoader)
	if !ok {
		return ""
	}

	opt, ok := pl.he.OptionsMap()[key]
	if !ok {
		return ""
	}

	return opt.String()
}
