package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/spf13/cobra"
)

func main() {
	cmd := newCommand()
	err := cmd.Execute()
	if err != nil {
		fmt.Println(cmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

var (
	logLevel         string
	addr             string
	config           string
	sslKey           string
	sslCert          string
	clientCA         string
	clientCertVerify bool
	cmdOpts          hiera.CommandOptions
	port             int
)

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "server",
		Short:  `Server - Start a Hiera REST server`,
		Long:   "Server - Start a REST server that performs lookups in a Hiera data storage.\n  Responds to key lookups under the /lookup endpoint",
		PreRun: initialize,
		Run:    startServer,
		Args:   cobra.NoArgs}

	flags := cmd.Flags()
	flags.StringVar(&logLevel, `loglevel`, `error`, `error/warn/info/debug`)
	flags.StringVar(&config, `config`, `/hiera/hiera.yaml`, `path to the hiera config file. Overrides /hiera/hiera.yaml`)
	flags.StringArrayVar(&cmdOpts.VarPaths, `vars`, nil, `path to a JSON or YAML file that contains key-value mappings to become variables for this lookup`)
	flags.StringArrayVar(&cmdOpts.Variables, `var`, nil, `variable as a key:value or key=value where value is a literal expressed in Puppet DSL`)
	flags.StringVar(&addr, `addr`, ``, `ip address to listen on`)
	flags.StringVar(&sslKey, `ssl-key`, ``, `ssl private key`)
	flags.StringVar(&sslCert, `ssl-cert`, ``, `ssl certificate`)
	flags.StringVar(&clientCA, `ca`, ``, `certificate authority to use to verify clients`)
	flags.BoolVar(&clientCertVerify, `clientCertVerify`, false, `verify client certificate`)
	flags.IntVar(&port, `port`, 8080, `port number to listen to`)
	return cmd
}

func initialize(_ *cobra.Command, _ []string) {
	issue.IncludeStacktrace(logLevel == `debug`)
}

var keyPattern = regexp.MustCompile(`^/lookup/(.*)$`)

func startServer(cmd *cobra.Command, _ []string) {
	configOptions := map[string]px.Value{
		provider.LookupKeyFunctions: types.WrapRuntime([]hieraapi.LookupKey{provider.ConfigLookupKey, provider.Environment})}

	configOptions[hieraapi.HieraConfig] = types.WrapString(config)

	hiera.DoWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(ctx px.Context) {
		ctx.Set(`logLevel`, px.LogLevelFromString(logLevel))
		router := CreateRouter(ctx)

		server := &http.Server{
			Addr:    addr + ":" + strconv.Itoa(port),
			Handler: router,
		}

		var err error
		var tlsConfig *tls.Config
		tlsConfig, err = makeTLSconfig()
		if err != nil {
			panic(err)
		}

		if tlsConfig == nil {
			err = server.ListenAndServe()
		} else {
			server.TLSConfig = tlsConfig
			err = server.ListenAndServeTLS("", "")
		}
		if err != nil {
			panic(err)
		}
	})
}

func CreateRouter(ctx px.Context) http.Handler {
	doLookup := func(w http.ResponseWriter, r *http.Request) {
		ks := keyPattern.FindStringSubmatch(r.URL.Path)
		if ks == nil {
			http.NotFound(w, r)
			return
		}
		key := ks[1]

		defer func() {
			if r := recover(); r != nil {
				var err error
				if er, ok := r.(error); ok {
					err = er
				} else if es, ok := r.(string); ok {
					err = errors.New(es)
				} else {
					panic(r)
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()

		opts := cmdOpts
		params := r.URL.Query()
		if dflt, ok := params[`default`]; ok && len(dflt) > 0 {
			opts.Default = &dflt[0]
		}
		opts.Merge = params.Get(`merge`)
		opts.Type = params.Get(`type`)
		opts.Variables = append(opts.Variables, params[`var`]...)
		opts.RenderAs = `json`
		out := bytes.Buffer{}
		if hiera.LookupAndRender(ctx, &opts, []string{key}, &out) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(out.Bytes())
		} else {
			http.Error(w, `404 value not found`, http.StatusNotFound)
		}
	}

	router := http.NewServeMux()
	router.HandleFunc("/lookup/", doLookup)
	return router
}

func loadCertPool(pemFile string) (*x509.CertPool, error) {
	data, err := ioutil.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(data)
	if !ok {
		return nil, fmt.Errorf("Failed to load certificate %s", pemFile)
	}

	return certPool, nil
}

func makeTLSconfig() (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	if sslCert == "" || sslKey == "" {
		return tlsConfig, nil
	}

	cert, err := tls.LoadX509KeyPair(sslCert, sslKey)
	if err != nil {
		return tlsConfig, err
	}

	tlsConfig.Certificates = []tls.Certificate{cert}

	if clientCertVerify {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if clientCA != "" {
		certPool, err := loadCertPool(clientCA)
		if err != nil {
			return tlsConfig, err
		}

		tlsConfig.ClientCAs = certPool
	}

	return tlsConfig, nil
}
