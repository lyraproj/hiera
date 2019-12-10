package cli

import (
	"bytes"

	"github.com/lyraproj/hiera/hiera"
)

// ExecuteLookup performs a lookup using the CLI. It's primarily intended for testing purposes
func ExecuteLookup(args ...string) (output []byte, err error) {
	cmdOpts = hiera.CommandOptions{}
	dflt = OptString{}
	logLevel = ``
	configPath = ``
	facts = nil

	cmd := NewCommand()
	buf := new(bytes.Buffer)
	cmd.SetOutput(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return buf.Bytes(), err
}
