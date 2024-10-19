package subcmds

import (
	"fmt"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/pysugar/wheels/uuid"
	"github.com/spf13/cobra"
)

var randStrCmd = &cobra.Command{
	Use:   `rand [-n 64]`,
	Short: "Generate Rand String",
	Long: `
Generate Rand String.

Rand String: netool rand -n 32
`,
	Run: func(cmd *cobra.Command, args []string) {
		n, _ := cmd.Flags().GetInt("num")
		var output string
		if n <= 0 {
			output = "the num must greater than 0"
		} else {
			output = uuid.GenerateRandomString(n)
		}
		fmt.Println(output)
	},
}

func init() {
	randStrCmd.Flags().IntP("num", "n", 32, "rand string length")
	base.AddSubCommands(randStrCmd)
}
