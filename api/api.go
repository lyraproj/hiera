// Package api contains interfaces that are used throughout the hiera code base
package api

// HieraRoot is an option key that can be used to change the default root which is the current working directory
const HieraRoot = `Hiera::Root`

// HieraConfigFileName is an option that can be used to change the default file name 'hiera.yaml'
const HieraConfigFileName = `Hiera::ConfigFileName`

// HieraConfig is an option that can be used to change absolute path of the hiera config. When specified, the
// HieraRoot and HieraConfigFileName will not have any effect.
const HieraConfig = `Hiera::Config`

// HieraDialect is an option that can be used to control the dialect of the type parser and streaming
// capabilities of Hiera. Valid values are "dgo" or "pcore".
const HieraDialect = `Hiera::Dialect`

// HieraScope is an option that can be used to pass a variable scope to Hiera. This scope is used
// by the 'scope' lookup_key provider function and when doing variable interpolations
const HieraScope = `Hiera::Scope`

// HieraFunctions is an option that can be used to pass custom lookup functions to Hiera. The value must
// be a dgo.Map with String keys and Function values.
const HieraFunctions = `Hiera::Functions`
