package fositeadapter

import (
	"crypto/rand"
	"fmt"
	"math/big"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

const userCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateDeviceCodes() (string, string, error) {
	deviceCode, err := randomB64(32)
	if err != nil {
		return "", "", err
	}
	userCode, err := generateUserCode()
	if err != nil {
		return "", "", err
	}
	return deviceCode, userCode, nil
}

func generateUserCode() (string, error) {
	code := make([]byte, 8)
	limit := big.NewInt(int64(len(userCodeAlphabet)))
	for i := range code {
		value, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", fmt.Errorf("read cryptographic randomness: %w", err)
		}
		code[i] = userCodeAlphabet[value.Int64()]
	}
	return string(code[:4]) + "-" + string(code[4:]), nil
}

func normalizeUserCode(value string) string {
	code := make([]byte, 0, 8)
	for _, r := range value {
		if r == '-' || r == ' ' {
			continue
		}
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		code = append(code, byte(r))
	}
	if len(code) != 8 {
		return ""
	}
	for _, r := range code {
		if !containsUserCodeByte(r) {
			return ""
		}
	}
	return string(code[:4]) + "-" + string(code[4:])
}

func containsUserCodeByte(value byte) bool {
	for i := range len(userCodeAlphabet) {
		if userCodeAlphabet[i] == value {
			return true
		}
	}
	return false
}

func deviceCodeHash(key []byte, raw string) []byte {
	return idpstore.HashSecret(key, "tinyidp/device-code/v1\x00"+raw)
}

func userCodeHash(key []byte, raw string) []byte {
	return idpstore.HashSecret(key, "tinyidp/user-code/v1\x00"+normalizeUserCode(raw))
}
