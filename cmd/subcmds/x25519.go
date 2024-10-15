package subcmds

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/curve25519"
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
		Run: executeX25519,
	}
)

func init() {
	x25519Cmd.Flags().StringP("input", "i", "", "base64 input")
	base.AddSubCommands(x25519Cmd)
}

func executeX25519(cmd *cobra.Command, args []string) {
	inputBase64, _ := cmd.Flags().GetString("input")

	privateKey, err := getPrivateKey(inputBase64)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Calculate public key
	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		fmt.Printf("Error calculating public key: %v\n", err)
		return
	}

	// Output private and public keys
	fmt.Printf("Private key: %v\nPublic key: %v\n",
		base64.RawURLEncoding.EncodeToString(privateKey),
		base64.RawURLEncoding.EncodeToString(publicKey))
}

func getPrivateKey(inputBase64 string) ([]byte, error) {
	var privateKey []byte
	var err error

	if len(inputBase64) > 0 {
		privateKey, err = base64.RawURLEncoding.DecodeString(inputBase64)
		if err != nil {
			return nil, fmt.Errorf("error decoding private key: %v", err)
		}
		if len(privateKey) != curve25519.ScalarSize {
			return nil, fmt.Errorf("invalid length of private key")
		}
	} else {
		// Generate random private key if not provided
		privateKey = make([]byte, curve25519.ScalarSize)
		if _, err = rand.Read(privateKey); err != nil {
			return nil, fmt.Errorf("error generating private key: %v", err)
		}

		// Modify private key as per algorithm described at https://cr.yp.to/ecdh.html.
		privateKey[0] &= 248  // Clear the lowest 3 bits
		privateKey[31] &= 127 // Clear the highest bit
		privateKey[31] |= 64  // Set the second highest bit
	}

	return privateKey, nil
}
