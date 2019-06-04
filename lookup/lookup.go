package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lyraproj/hiera/explain"

	"github.com/lyraproj/pcore/yaml"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/internal"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/spf13/cobra"
	yl "gopkg.in/yaml.v3"
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

var (
	logLevel       string
	merge          string
	facts          string
	dflt           string
	typ            string
	renderAs       string
	explainData    bool
	explainOptions bool
)

func main() {
	cmd := newCommnand()
	err := cmd.Execute()
	if err != nil {
		fmt.Println(cmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

func newCommnand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lookup <key> [<key> ...]",
		Short:   `Lookup - Perform lookups in Hiera data storage`,
		Long:    "Lookup - Perform lookups in Hiera data storage.\n  Find more information at: https://github.com/lyraproj/hiera",
		Version: fmt.Sprintf("%v", getVersion()),
		PreRun:  initialize,
		Run:     lookup,
		Args:    cobra.MinimumNArgs(1)}

	flags := cmd.Flags()
	flags.StringVar(&logLevel, `loglevel`, `error`, `error/warn/info/debug`)
	flags.StringVar(&merge, `merge`, `first`, `first/unique/hash/deep`)
	flags.StringVar(&facts, `facts`, ``, `path to a JSON or YAML file that contains key-value mappings to become facts for this lookup`)
	flags.StringVar(&dflt, `default`, ``, `a value to return if Hiera can't find a value in data`)
	flags.StringVar(&typ, `type`, `Any`, `assert that the value has the specified type`)
	flags.StringVar(&renderAs, `render-as`, ``, `s/json/yaml/binary: Specify the output format of the results; s means plain text`)
	flags.BoolVar(&explainData, `explain`, false, `Explain the details of how the lookup was performed and where the final value came from (or the reason no value was found)`)
	flags.BoolVar(&explainOptions, `explain-options`, false, `Explain whether a lookup_options hash affects this lookup, and how that hash was assembled`)

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

func lookup(cmd *cobra.Command, args []string) {
	configOptions := map[string]px.Value{
		provider.LookupProvidersKey: types.WrapRuntime([]hieraapi.LookupKey{provider.ConfigLookupKey, provider.Environment})}

	hiera.DoWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(c px.Context) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(cmd.OutOrStderr(), r)
				os.Exit(1)
			}
		}()

		var tp px.Type = types.DefaultAnyType()
		if typ != `` {
			tp = c.ParseType(typ)
		}

		options := make(map[string]px.Value)
		if merge != `first` {
			options[`merge`] = types.WrapString(merge)
		}

		var dv px.Value
		if dflt != `` {
			if !tp.Equals(types.DefaultAnyType(), nil) {
				dv = types.CoerceTo(c, `default value`, tp, types.ParseFile(`<default value>`, dflt))
			} else {
				dv = types.WrapString(dflt)
			}
		}

		scope := px.EmptyMap
		if facts != `` {
			content := types.BinaryFromFile(facts)
			yv := yaml.Unmarshal(c, content.Bytes())
			if data, ok := yv.(px.OrderedMap); ok {
				scope = data
			} else {
				panic(px.Error(hieraapi.YamlNotHash, issue.H{`path`: facts}))
			}
		}

		var explainer explain.Explainer
		if explainData || explainOptions {
			if renderAs != `` {
				var ex string
				if explainData {
					ex = `explain`
				} else {
					ex = `explain-options`
				}
				panic(fmt.Errorf(`--render-as is mutually exclusive to --%s`, ex))
			}
			explainer = explain.NewExplainer(explainOptions, explainOptions && !explainData)
		}

		found := hiera.Lookup2(internal.NewInvocation(c, scope, explainer), args, tp, dv, nil, nil, options, nil)
		if explainer != nil {
			cmd.Println(explainer)
			return
		}

		if found == nil {
			os.Exit(1)
		}

		if renderAs == `` {
			renderAs = `yaml`
		}

		out := cmd.OutOrStdout()
		switch renderAs {
		case `yaml`, `json`:
			var v interface{}
			if !found.Equals(px.Undef, nil) {
				rf := c.Reflector().Reflect(found)
				if rf.IsValid() && rf.CanInterface() {
					v = rf.Interface()
				} else {
					v = found.String()
				}
			}
			var bs []byte
			var err error
			if renderAs == `yaml` {
				bs, err = yl.Marshal(v)
			} else {
				bs, err = json.Marshal(v)
			}
			if err != nil {
				panic(err)
			}
			utils.WriteString(out, string(bs))
		case `binary`:
			bi := types.CoerceTo(c, `lookup value`, types.DefaultBinaryType(), found).(*types.Binary)
			_, err := out.Write(bi.Bytes())
			if err != nil {
				panic(err)
			}
		case `s`:
			cmd.Println(found)
		}
	})
}
