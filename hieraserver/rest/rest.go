package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	logLevel string
	config   string
	cmdOpts  hiera.CommandOptions
	port     int
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
		err := http.ListenAndServe(":"+strconv.Itoa(port), router)
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
