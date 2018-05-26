package lookup

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

type Config interface {

}

type Entry interface {

}

type LookupKind string

type function struct {
	kind LookupKind
	name string
}

type defaults struct {
	options *types.HashValue
	dataDir string
	function function
}

type DataProvider interface {

}

type config struct {
	configRoot string
	configPath string
	loadedConfig *types.HashValue
	config *types.HashValue
	dataProviders []DataProvider
}

type entry struct {
	options *types.HashValue
	dataDir string
	function function
}

var hieraTypeSet eval.TypeSet

func init() {
	hieraTypeSet = eval.NewTypeSet(`Hiera`, `{
		pcore_version => '1.0.0',
		version => '5.0.0',
		types => {
			Options => Hash[Pattern[/\A[A-Za-z](:?[0-9A-Za-z_-]*[0-9A-Za-z])?\z/], Data],
			Defaults => Struct[{
				Optional[options] => Options,
				Optional[data_dig] => String[1],
				Optional[data_hash] => String[1],
				Optional[lookup_key] => String[1],
				Optional[data_dir] => String[1],
			}],
			Entry => Struct[{
				name => String[1],
				Optional[options] => Options,
				Optional[data_dig] => String[1],
				Optional[data_hash] => String[1],
				Optional[lookup_key] => String[1],
				Optional[data_dir] => String[1],
				Optional[path] => String[1],
				Optional[paths] => Array[String[1], 1],
				Optional[glob] => String[1],
				Optional[globs] => Array[String[1], 1],
				Optional[uri] => String[1],
				Optional[uris] => Array[String[1], 1],
				Optional[mapped_paths] => Array[String[1], 3, 3],
			}],
			Config => Struct[{
				version => Integer[5, 5],
				Optional[defaults] => Defaults,
				Optional[hierarchy] => Array[Entry],
				Optional[default_hierarchy] => Array[Entry]
			}]
		}
  }`)
}

func newConfig(c eval.Context, initHash *types.HashValue) Config {
	return nil
}