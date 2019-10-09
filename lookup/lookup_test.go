package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookup_defaultInt(t *testing.T) {
	result, err := executeLookup(`--default`, `23`, `--type`, `Integer`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "23\n", string(result))
}

func TestLookup_defaultString(t *testing.T) {
	result, err := executeLookup(`--default`, `23`, `--type`, `String`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "\"23\"\n", string(result))
}

func TestLookup_defaultEmptyString(t *testing.T) {
	result, err := executeLookup(`--default`, ``, `foo`)
	require.NoError(t, err)
	require.Equal(t, "\"\"\n", string(result))
}

func TestLookup_defaultHash(t *testing.T) {
	result, err := executeLookup(`--default`, `{ x => 'a', y => 9 }`, `--type`, `Hash[String,Variant[String,Integer]]`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "x: a\ny: 9\n", string(result))
}

func TestLookup_defaultHash_json(t *testing.T) {
	result, err := executeLookup(`--default`, `{ x => 'a', y => 9 }`, `--type`, `Hash[String,Variant[String,Integer]]`, `--render-as`, `json`, `foo`)
	require.NoError(t, err)
	require.Equal(t, `{"x":"a","y":9}`, string(result))
}

func TestLookup_defaultString_s(t *testing.T) {
	result, err := executeLookup(`--default`, `xyz`, `--render-as`, `s`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "xyz\n", string(result))
}

func TestLookup_defaultString_binary(t *testing.T) {
	result, err := executeLookup(`--default`, `YWJjMTIzIT8kKiYoKSctPUB+`, `--render-as`, `binary`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "abc123!?$*&()'-=@~", string(result))
}

func TestLookup_defaultArray_binary(t *testing.T) {
	result, err := executeLookup(`--default`, `[12, 28, 37, 15]`, `--type`, `Array[Integer]`, `--render-as`, `binary`, `foo`)
	require.NoError(t, err)
	require.Equal(t, []byte{12, 28, 37, 15}, result)
}

func TestLookup_facts(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--facts`, `facts.yaml`, `interpolate_a`)
		require.NoError(t, err)
		require.Equal(t, "This is value of a\n", string(result))
	})
}

func TestLookup_fact_interpolated_config(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--facts`, `facts.yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Equal(t, "This is value of c.a\n", string(result))
	})
}

func TestLookup_vars_interpolated_config(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--vars`, `facts.yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Equal(t, "This is value of c.a\n", string(result))
	})
}

func TestLookup_var_interpolated_config(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--var`, `c={a=>'the option value'}`, `--var`, `data_file: by_fact`, `interpolate_ca`)
		require.NoError(t, err)
		require.Equal(t, "This is the option value\n", string(result))
	})
}

func TestLookup_fact_directly(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--facts`, `facts.yaml`, `--config`, `fact_directly.yaml`, `the_fact`)
		require.NoError(t, err)
		require.Equal(t, "value of the_fact\n", string(result))
	})
}

func TestLookup_nullentry(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`nullentry`)
		require.NoError(t, err)
		require.Equal(t, "nv: null\n", string(result))
	})
}

func TestLookup_explain(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--explain`, `--facts`, `facts.yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Regexp(t,
			`\ASearching for "interpolate_ca"
  Merge strategy "first found strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        path not found
        No such key: "interpolate_ca"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}.yaml"
        Interpolation on "This is %\{c\.a\}"
          Sub key: "a"
            Found key: "a" value: 'value of c.a'
        Found key: "interpolate_ca" value: 'This is value of c\.a'
    Merged result: 'This is value of c\.a'
\z`, string(result))
	})
}

func TestLookup_explain_yaml(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--explain`, `--facts`, `facts.yaml`, `--render-as`, `yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Regexp(t,
			`\A__ptype: Hiera::Explainer
branches:
  - __ptype: Hiera::ExplainLookup
    branches:
      - __ptype: Hiera::ExplainMerge
        branches:
          - __ptype: Hiera::ExplainProvider
            branches:
              - __ptype: Hiera::ExplainLocation
                event: 5
                key: interpolate_ca
                location:
                    __ptype: Hiera::Path
                    exists: true
                    original: common\.yaml
                    resolved: .*/testdata/hiera/common\.yaml
            providerName: data_hash function 'yaml_data'
          - __ptype: Hiera::ExplainProvider
            branches:
              - __ptype: Hiera::ExplainLocation
                branches:
                  - __ptype: Hiera::ExplainInterpolate
                    branches:
                      - __ptype: Hiera::ExplainSubLookup
                        branches:
                          - __ptype: Hiera::ExplainKeySegment
                            event: 1
                            key: a
                            segment: a
                            value: value of c\.a
                        subKey: c\.a
                    expression: This is %\{c\.a\}
                event: 1
                key: interpolate_ca
                location:
                    __ptype: Hiera::Path
                    exists: true
                    original: named_%\{data_file\}\.yaml
                    resolved: .*/testdata/hiera/named_by_fact\.yaml
                value: This is value of c\.a
            providerName: data_hash function 'yaml_data'
        event: 6
        strategy: first
        value: This is value of c\.a
    key: interpolate_ca
\z`, string(result))
	})
}

func TestLookup_explain_options(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--explain-options`, `--facts`, `facts.yaml`, `hash`)
		require.NoError(t, err)
		require.Regexp(t,
			`\ASearching for "lookup_options"
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        Found key: "lookup_options" value: \{
          'hash' => \{
            'merge' => 'deep'
          \},
          'sense' => \{
            'convert_to' => 'Sensitive'
          \}
        \}
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}\.yaml"
        path not found
        No such key: "lookup_options"
    Merged result: \{
        'hash' => \{
          'merge' => 'deep'
        \},
        'sense' => \{
          'convert_to' => 'Sensitive'
        \}
      \}
\z`, string(result))
	})
}

func TestLookup_explain_explain_options(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--explain`, `--explain-options`, `--facts`, `facts.yaml`, `hash`)
		require.NoError(t, err)
		require.Regexp(t,
			`\ASearching for "lookup_options"
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        Found key: "lookup_options" value: \{
          'hash' => \{
            'merge' => 'deep'
          \},
          'sense' => \{
            'convert_to' => 'Sensitive'
          \}
        \}
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}\.yaml"
        path not found
        No such key: "lookup_options"
    Merged result: \{
        'hash' => \{
          'merge' => 'deep'
        \},
        'sense' => \{
          'convert_to' => 'Sensitive'
        \}
      \}
Searching for "hash"
  Using merge options from "lookup_options" hash
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        Found key: "hash" value: \{
          'one' => 1,
          'two' => 'two',
          'three' => \{
            'a' => 'A',
            'c' => 'C'
          \}
        \}
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}\.yaml"
        Found key: "hash" value: \{
          'one' => 'overwritten one',
          'three' => \{
            'a' => 'overwritten A',
            'b' => 'B',
            'c' => 'overwritten C'
          \}
        \}
    Merged result: \{
        'one' => 1,
        'two' => 'two',
        'three' => \{
          'a' => 'A',
          'c' => 'C',
          'b' => 'B'
        \}
      \}
\z`, string(result))
	})
}

func customLK(hc hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
	if v, ok := options[key]; ok {
		return v
	}
	hc.NotFound()
	return nil // Not reached
}

func init() {
	px.NewGoFunction(`customLK`,
		func(d px.Dispatch) {
			d.Param(`Hiera::Context`)
			d.Param(`String`)
			d.Param(`Hash[String,Any]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return customLK(args[0].(hieraapi.ProviderContext), args[1].String(), args[2].(px.OrderedMap).ToStringMap())
			})
		},
	)
}

func TestLookup_withCustomLK(t *testing.T) {

	inTestdata(func() {
		result, err := executeLookup(`--config`, `with_custom_provider.yaml`, `a`)
		require.NoError(t, err)
		require.Equal(t, "option a\n", string(result))
	})
}

func TestLookup_TerraformBackend(t *testing.T) {
	inTestdata(func() {
		result, err := executeLookup(`--var`, `backend:local`, `--var`, `path:terraform.tfstate`, `--config`, `terraform_backend.yaml`, `test`)
		require.NoError(t, err)
		require.Equal(t, "value\n", string(result))
	})
	inTestdata(func() {
		result, err := executeLookup(`--var`, `backend:local`, `--var`, `path:terraform.tfstate`, `--config`, `terraform_backend.yaml`, `--render-as`, `json`, `testobject`)
		require.NoError(t, err)
		require.Equal(t, `{"key1":"value1","key2":"value2"}`, string(result))
	})
}

func TestLookup_TerraformBackendErrors(t *testing.T) {
	inTestdata(func() {
		_, err := executeLookup(`--var`, `backend:something`, `--config`, `terraform_backend.yaml`, `test`)
		if assert.Error(t, err) {
			require.Regexp(t, `Unknown backend type "something"`, err.Error())
		}
	})
	inTestdata(func() {
		_, err := executeLookup(`--var`, `backend:local`, `--var`, `path:something`, `--config`, `terraform_backend.yaml`, `test`)
		if assert.Error(t, err) {
			require.Regexp(t, `RootModule called on nil State`, err.Error())
		}
	})
	inTestdata(func() {
		_, err := executeLookup(`--var`, `backend:local`, `--config`, `terraform_backend_errors.yaml`, `test`)
		if assert.Error(t, err) {
			require.Regexp(t, `The given configuration is not valid for backend "local"`, err.Error())
		}
	})
}

func inTestdata(f func()) {
	cw, err := os.Getwd()
	if err == nil {
		err = os.Chdir(`testdata`)
		if err == nil {
			defer func() {
				_ = os.Chdir(cw)
			}()
			f()
		}
	}
	if err != nil {
		panic(err)
	}
}

func executeLookup(args ...string) (output []byte, err error) {
	cmd := newCommand()
	buf := new(bytes.Buffer)
	cmd.SetOutput(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return buf.Bytes(), err
}
