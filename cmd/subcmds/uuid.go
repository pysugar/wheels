package subcmds

import (
	"fmt"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/pysugar/wheels/uuid"
	"github.com/spf13/cobra"
)

var uuidCmd = &cobra.Command{
	Use:   `uuid [-i "example"]`,
	Short: "Generate UUIDv4 or UUIDv5",
	Long: `
Generate UUIDv4 or UUIDv5.

UUIDv4 (random): netool uuid

UUIDv5 (from input): netool uuid -i "example"
`,
	Run: func(cmd *cobra.Command, args []string) {
		input, _ := cmd.Flags().GetString("input")
		var output string
		if l := len(input); l == 0 {
			u := uuid.New()
			output = u.String()
		} else if l <= 30 {
			u, _ := uuid.ParseString(input)
			output = u.String()
		} else {
			output = "Input must be within 30 bytes."
		}
		fmt.Println(output)
	},
}

func init() {
	uuidCmd.Flags().StringP("input", "i", "example", "seed")
	base.AddSubCommands(uuidCmd)
}
