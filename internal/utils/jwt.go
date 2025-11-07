package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateAccessToken returns a cryptographically random token string.
func GenerateAccessToken(claims map[string]interface{}) (string, error) {
	return generateOpaqueToken("access", claims)
}

// GenerateRefreshToken returns a cryptographically random token string.
func GenerateRefreshToken(claims map[string]interface{}) (string, error) {
	return generateOpaqueToken("refresh", claims)
}

func generateOpaqueToken(prefix string, _ map[string]interface{}) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", prefix, base64.RawURLEncoding.EncodeToString(buf)), nil
}
