package main

import (
	"bytes"
	"os"
	"testing"

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
        Original path: "named_%{data_file}.yaml"
        Interpolation on "This is %\{c\.a\}"
          Sub key: "a"
            Found key: "a" value: 'value of c.a'
        Found key: "interpolate_ca" value: 'This is value of c\.a'
    Merged result: 'This is value of c\.a'
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
	cmd := newCommnand()
	buf := new(bytes.Buffer)
	cmd.SetOutput(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return buf.Bytes(), err
}
