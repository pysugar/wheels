package signature

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"log"
)

func VerifySignature(payload, key, signature []byte) bool {
	targetSignature, err := Sign(payload, key)
	log.Printf("target: %s, expect: %s\n", targetSignature, signature)
	if err != nil {
		log.Printf("unexpected sign error: %v", err)
		return false
	}
	return bytes.Equal(signature, targetSignature)
}

func Sign(payload, key []byte) ([]byte, error) {
	hmacKey := hmac.New(sha256.New, key)
	if _, err := hmacKey.Write(payload); err != nil {
		return nil, err
	}

	hmacSum := hmacKey.Sum(nil)
	base64Len := base64.StdEncoding.EncodedLen(len(hmacSum))
	base64Result := make([]byte, base64Len)
	base64.StdEncoding.Encode(base64Result, hmacSum)

	return base64Result, nil
}
