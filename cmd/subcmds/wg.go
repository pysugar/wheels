package subcmds

import (
	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
)

var (
	wgCmd = &cobra.Command{
		Use:   `wg [-i "private key (base64.StdEncoding)"]`,
		Short: "Generate key pair for wireguard key exchange",
		Long: `
Generate key pair for wireguard key exchange.

Random: netool wg

From private key: netool wg -i "private key (base64.StdEncoding)"
`,
		Run: func(cmd *cobra.Command, args []string) {
			wgInput, _ := cmd.Flags().GetString("input")
			Curve25519GenKey(true, wgInput)
		},
	}
)

func init() {
	wgCmd.Flags().StringP("input", "i", "", "wireguard input")
	base.AddSubCommands(wgCmd)
}
