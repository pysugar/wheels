package subcmds

import (
	"encoding/base64"
	"fmt"
	"github.com/pysugar/wheels/authenticate/signature"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
)

var (
	signatureCmd = &cobra.Command{
		Use:   `signature`,
		Short: "Signature Commands",
		Long: `
Signature Commands.

Signature Commands: netool signature --help
`,
		Run: func(cmd *cobra.Command, args []string) {
			k, _ := cmd.Flags().GetString("key")
			i, _ := cmd.Flags().GetString("input")

			key, err := base64.StdEncoding.DecodeString(k)
			if err != nil {
				fmt.Printf("invalid key: %v", err)
				return
			}
			input, err := base64.StdEncoding.DecodeString(i)
			if err != nil {
				fmt.Printf("invalid input: %v", err)
				return
			}

			sign, err := signature.Sign(input, key)
			if err != nil {
				fmt.Printf("sign error: %v", err)
				return
			}
			fmt.Printf("sign result: %s\n", string(sign))
		},
	}

	signatureVerifyCmd = &cobra.Command{
		Use: `verify`,
		Run: func(cmd *cobra.Command, args []string) {
			k, _ := cmd.Flags().GetString("key")
			i, _ := cmd.Flags().GetString("input")
			s, _ := cmd.Flags().GetString("signature")

			key, err := base64.StdEncoding.DecodeString(k)
			if err != nil {
				fmt.Printf("invalid key: %v", err)
				return
			}
			input, err := base64.StdEncoding.DecodeString(i)
			if err != nil {
				fmt.Printf("invalid input: %v", err)
				return
			}
			sign, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				fmt.Printf("invalid signature: %v", err)
				return
			}

			fmt.Printf("verify result: %v\n", signature.VerifySignature(input, key, sign))
		},
	}
)

func init() {
	signatureCmd.Flags().StringP("key", "k", "", "base64 secret key")
	signatureCmd.Flags().StringP("input", "i", "", "base64 input")

	signatureVerifyCmd.Flags().StringP("key", "k", "", "base64 secret key")
	signatureVerifyCmd.Flags().StringP("input", "i", "", "base64 input")
	signatureVerifyCmd.Flags().StringP("signature", "s", "", "expect base64 signature")
	signatureCmd.AddCommand(signatureVerifyCmd)
	base.AddSubCommands(signatureCmd)
}
