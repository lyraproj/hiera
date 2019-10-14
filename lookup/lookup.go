package main

import (
	"fmt"
	"os"

	"github.com/lyraproj/hiera/cli"
)

func main() {
	cmd := cli.NewCommand()
	err := cmd.Execute()
	if err != nil {
		fmt.Println(cmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
