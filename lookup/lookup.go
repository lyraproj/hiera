package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/issue/issue"
	"github.com/spf13/cobra"
)

var helpTemplate = `Description:
  {{rpad .Long 10}}

Usage:{{if .Runnable}}{{if .HasAvailableFlags}}
  {{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample }}

Examples:
  {{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}
`

type OptString struct {
	value *string
}

func (s *OptString) Type() string {
	return "stringpointer"
}

func (s *OptString) String() string {
	if s == nil || s.value == nil {
		return ``
	}
	return *s.value
}

func (s *OptString) Set(v string) error {
	s.value = &v
	return nil
}

func (s *OptString) StringPointer() *string {
	return s.value
}

var (
	cmdOpts  hiera.CommandOptions
	dflt     OptString
	logLevel string
	config   string
)

func main() {
	cmd := newCommand()
	err := cmd.Execute()
	if err != nil {
		fmt.Println(cmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lookup <key> [<key> ...]",
		Short:   `Lookup - Perform lookups in Hiera data storage`,
		Long:    "Lookup - Perform lookups in Hiera data storage.\n  Find more information at: https://github.com/lyraproj/hiera",
		Version: fmt.Sprintf("%v", getVersion()),
		PreRun:  initialize,
		Run:     cmdLookup,
		Args:    cobra.MinimumNArgs(1)}

	flags := cmd.Flags()
	flags.StringVar(&logLevel, `loglevel`, `error`, `error/warn/info/debug`)
	flags.StringVar(&cmdOpts.Merge, `merge`, `first`, `first/unique/hash/deep`)
	flags.StringVar(&config, `config`, ``, `path to the hiera config file. Overrides <current directory>/hiera.yaml`)
	flags.StringVar(&cmdOpts.Variables, `facts`, ``, `path to a JSON or YAML file that contains key-value mappings to become facts for this lookup`)
	flags.Var(&dflt, `default`, `a value to return if Hiera can't find a value in data`)
	flags.StringVar(&cmdOpts.Type, `type`, `Any`, `assert that the value has the specified type`)
	flags.StringVar(&cmdOpts.RenderAs, `render-as`, ``, `s/json/yaml/binary: Specify the output format of the results; s means plain text`)
	flags.BoolVar(&cmdOpts.ExplainData, `explain`, false, `Explain the details of how the lookup was performed and where the final value came from (or the reason no value was found)`)
	flags.BoolVar(&cmdOpts.ExplainOptions, `explain-options`, false, `Explain whether a lookup_options hash affects this lookup, and how that hash was assembled`)

	cmd.SetHelpTemplate(helpTemplate)
	return cmd
}

func initialize(_ *cobra.Command, _ []string) {
	issue.IncludeStacktrace(logLevel == `debug`)
	hclog.DefaultOptions = &hclog.LoggerOptions{
		Name:  `lookup`,
		Level: hclog.LevelFromString(logLevel),
	}
}

func cmdLookup(cmd *cobra.Command, args []string) {
	cmdOpts.Default = dflt.StringPointer()
	configOptions := map[string]px.Value{
		provider.LookupProvidersKey: types.WrapRuntime([]hieraapi.LookupKey{provider.ConfigLookupKey, provider.Environment})}

	if config != `` {
		configOptions[hieraapi.HieraConfig] = types.WrapString(config)
	}

	err := hiera.TryWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(c px.Context) error {
		hiera.LookupAndRender(c, &cmdOpts, args, cmd.OutOrStdout())
		return nil
	})
	if err != nil {
		fmt.Println(cmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
