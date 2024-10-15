package subcmds

import (
	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
)

var (
	x25519Cmd = &cobra.Command{
		Use:   `x25519 [-i "private key (base64.RawURLEncoding)"]`,
		Short: "Generate key pair for x25519 key exchange",
		Long: `
Generate key pair for x25519 key exchange.

Random: netool x25519

From private key: netool x25519 -i "private key (base64.RawURLEncoding)"
`,
		Run: func(cmd *cobra.Command, args []string) {
			stdEncoding, _ := cmd.Flags().GetBool("std-encoding")
			inputBase64, _ := cmd.Flags().GetString("input")
			Curve25519GenKey(stdEncoding, inputBase64)
		},
	}
)

func init() {
	x25519Cmd.Flags().BoolP("std-encoding", "e", false, "")
	x25519Cmd.Flags().StringP("input", "i", "", "base64 input")
	base.AddSubCommands(x25519Cmd)
}
