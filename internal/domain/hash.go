package domain

import (
	"crypto/hmac"
	"crypto/sha256"
)

func HashSecret(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
