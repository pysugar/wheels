package subcmds

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/curve25519"
)

func Curve25519GenKey(stdEncoding bool, inputBase64 string) {
	var encoding *base64.Encoding

	if stdEncoding {
		encoding = base64.StdEncoding
	} else {
		encoding = base64.RawURLEncoding
	}

	privateKey, err := generatePrivateKey(inputBase64, encoding)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Calculate the public key
	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		fmt.Println("Error calculating public key:", err)
		return
	}

	// Output the private and public keys
	fmt.Printf("Private key: %s\nPublic key: %s\n",
		encoding.EncodeToString(privateKey),
		encoding.EncodeToString(publicKey))
}

func generatePrivateKey(inputBase64 string, encoding *base64.Encoding) ([]byte, error) {
	var privateKey []byte

	if len(inputBase64) > 0 {
		// Decode the provided base64 private key
		decodedKey, err := encoding.DecodeString(inputBase64)
		if err != nil {
			return nil, fmt.Errorf("error decoding private key: %v", err)
		}
		if len(decodedKey) != curve25519.ScalarSize {
			return nil, fmt.Errorf("invalid length of private key")
		}
		privateKey = decodedKey
	} else {
		// Generate a random private key
		privateKey = make([]byte, curve25519.ScalarSize)
		if _, err := rand.Read(privateKey); err != nil {
			return nil, fmt.Errorf("error generating private key: %v", err)
		}

		// Modify random bytes as per the algorithm specification
		// - The first byte must be cleared of its lowest 3 bits to make it a multiple of 8.
		// - The highest 3 bits of the last byte must be adjusted to fit into the Curve25519 specification.
		privateKey[0] &= 248
		privateKey[31] &= 127
		privateKey[31] |= 64
	}

	return privateKey, nil
}
