package cli

import (
	"context"
	"fmt"

	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/config"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/provider"
	sdk "github.com/lyraproj/hierasdk/hiera"
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

// OptString is a string option that can differentiate between an empty string and no value
type OptString struct {
	value *string
}

// Type of option
func (s *OptString) Type() string {
	return "stringpointer"
}

// String value
func (s *OptString) String() string {
	if s == nil || s.value == nil {
		return ``
	}
	return *s.value
}

// Set sets the string value
func (s *OptString) Set(v string) error {
	s.value = &v
	return nil
}

// StringPointer returns the interal value pointer
func (s *OptString) StringPointer() *string {
	return s.value
}

var (
	cmdOpts    hiera.CommandOptions
	dflt       OptString
	logLevel   string
	configPath string
	dialect    string
)

// NewCommand creates the hiera Command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lookup <key> [<key> ...]",
		Short: `MergeLookup - Perform lookups in Hiera data storage`,
		Long: `MergeLookup - Perform lookups in Hiera data storage.
    Find more information at: https://github.com/lyraproj/hiera`,
		Version: fmt.Sprintf("%v", getVersion()),
		RunE:    cmdLookup,
		Args:    cobra.MinimumNArgs(1)}

	flags := cmd.Flags()
	flags.StringVar(&logLevel, `loglevel`, `error`,
		`error/warn/info/debug`)
	flags.StringVar(&cmdOpts.Merge, `merge`, `first`,
		`first/unique/hash/deep`)
	flags.StringVar(&configPath, `config`, ``,
		`path to the hiera config file. Overrides <current directory>/`+config.FileName)
	flags.Var(&dflt, `default`,
		`a value to return if Hiera can't find a value in data`)
	flags.StringVar(&cmdOpts.Type, `type`, ``,
		`assert that the value has the specified type (if using --all this must be a map)`)
	flags.StringVar(&dialect, `dialect`, `pcore`,
		`dialect to use for rich data serialization and parsing of types pcore|dgo'`)
	flags.StringVar(&cmdOpts.RenderAs, `render-as`, ``,
		`s/json/yaml/binary: Specify the output format of the results; s means plain text`)
	flags.BoolVar(&cmdOpts.ExplainData, `explain`, false,
		`Explain the details of how the lookup was performed and where the final value came from`)
	flags.BoolVar(&cmdOpts.ExplainOptions, `explain-options`, false,
		`Explain whether a lookup_options hash affects this lookup, and how that hash was assembled`)
	flags.StringArrayVar(&cmdOpts.VarPaths, `vars`, nil,
		`path to a JSON or YAML file that contains key-value mappings to become variables for this lookup`)
	flags.StringArrayVar(&cmdOpts.Variables, `var`, nil,
		`a key:value or key=value where value is literal expressed using Puppet DSL`)
	flags.StringArrayVar(&cmdOpts.FactPaths, `facts`, nil,
		`like --vars but will also make variables available under the "facts" (for compatibility with Puppet's ruby version of Hiera)`)
	flags.BoolVar(&cmdOpts.LookupAll, `all`, false,
		`lookup all of the keys and output the results as a map`)

	cmd.SetHelpTemplate(helpTemplate)
	return cmd
}

func cmdLookup(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmdOpts.Default = dflt.StringPointer()
	cfgOpts := vf.MutableMap()
	cfgOpts.Put(api.HieraDialect, dialect)
	cfgOpts.Put(
		provider.LookupKeyFunctions, []sdk.LookupKey{provider.ConfigLookupKey, provider.Environment})

	if configPath != `` {
		cfgOpts.Put(api.HieraConfig, configPath)
	}

	return hiera.TryWithParent(context.Background(), provider.MuxLookupKey, cfgOpts, func(c api.Session) error {
		hiera.LookupAndRender(c, &cmdOpts, args, cmd.OutOrStdout())
		return nil
	})
}
