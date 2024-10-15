package main

import (
	"fmt"
	"github.com/spf13/cobra"

	"github.com/pysugar/wheels/cmd/base"
	_ "github.com/pysugar/wheels/cmd/subcmds"
)

var (
	versionCmd = &cobra.Command{
		Use:   `version`,
		Short: "Show current version of Netool",
		Long:  `Version prints the build information for Netool executables`,
		Run: func(cmd *cobra.Command, args []string) {
			version := base.VersionStatement()
			for _, s := range version {
				fmt.Println(s)
			}
		},
	}
)

func main() {
	base.AddSubCommands(versionCmd)

	base.Run()
}
