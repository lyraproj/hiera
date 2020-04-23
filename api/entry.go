package api

import (
	"github.com/lyraproj/dgo/dgo"
)

// An Entry is a definition an entry in the hierarchy.
type Entry interface {
	// Create a copy of this entry for the given Config
	Copy(Config) Entry

	// Options returns the options
	Options() dgo.Map

	// DataDir returns datadir
	DataDir() string

	// PluginDir returns plugindir
	PluginDir() string

	// PluginFile returns pluginfile
	PluginFile() string

	// Function returns data_dir, data_hash, or lookup_key function
	Function() Function

	// Name returns the name
	Name() string

	// Resolve resolves this configuration on behalf of the given invocation and defaults entry
	Resolve(ic Invocation, defaults Entry) Entry

	// Locations returns the paths, globs, or uris. The method returns nil if no locations are defined
	Locations() []Location
}
