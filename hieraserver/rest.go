package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"

	"github.com/hashicorp/go-hclog"
	"github.com/labstack/echo"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
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
		Long:   "Lookup - Start a REST server that performs lookups in a Hiera data storage.\n  Find more information at: https://github.com/lyraproj/hiera",
		PreRun: initialize,
		Run:    startServer,
		Args:   cobra.NoArgs}

	flags := cmd.Flags()
	flags.StringVar(&logLevel, `loglevel`, `error`, `error/warn/info/debug`)
	flags.StringVar(&config, `config`, ``, `path to the hiera config file. Overrides <current directory>/hiera.yaml`)
	flags.StringVar(&cmdOpts.Variables, `variables`, ``, `path to a JSON or YAML file that contains key-value mappings to become scope variables`)
	flags.IntVar(&port, `port`, 80, `port number to listen to`)
	return cmd
}

func initialize(_ *cobra.Command, _ []string) {
	issue.IncludeStacktrace(logLevel == `debug`)
	hclog.DefaultOptions = &hclog.LoggerOptions{
		Name:  `lookup`,
		Level: hclog.LevelFromString(logLevel),
	}
}

func startServer(cmd *cobra.Command, _ []string) {
	e := echo.New()
	e.Logger.SetOutput(cmd.OutOrStdout())
	configOptions := map[string]px.Value{
		provider.LookupProvidersKey: types.WrapRuntime([]hieraapi.LookupKey{provider.ConfigLookupKey, provider.Environment})}

	if config != `` {
		configOptions[hieraapi.HieraConfig] = types.WrapString(config)
	}

	hiera.DoWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(ctx px.Context) {
		doLookup := func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					if rp, ok := r.(issue.Reported); ok {
						err = c.JSON(http.StatusBadRequest, map[string]string{`message`: rp.Error()})
					} else if er, ok := r.(error); ok {
						err = er
					} else {
						panic(r)
					}
				}
			}()

			opts := cmdOpts
			key := c.Param(`key`)
			params := c.QueryParams()
			if dflt, ok := params[`default`]; ok && len(dflt) > 0 {
				opts.Default = &dflt[0]
			}
			opts.Merge = params.Get(`merge`)
			opts.Type = params.Get(`type`)
			opts.RenderAs = `json`
			out := bytes.Buffer{}
			if hiera.LookupAndRender(ctx, &opts, []string{key}, &out) {
				err = c.Stream(http.StatusOK, echo.MIMEApplicationJSONCharsetUTF8, bytes.NewBuffer(out.Bytes()))
			} else {
				err = c.NoContent(http.StatusNotFound)
			}
			return
		}

		e.GET("/lookup/:key", doLookup)
		e.Logger.Fatal(e.Start(":" + strconv.Itoa(port)))
	})
}
