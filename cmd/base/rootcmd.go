package base

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "netool",
	Short: "net tool",
	Long:  "A simple CLI for Net tool",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, this is a net tool")
	},
}

func AddSubCommands(cmds ...*cobra.Command) {
	rootCmd.AddCommand(cmds...)
}

func Run() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
