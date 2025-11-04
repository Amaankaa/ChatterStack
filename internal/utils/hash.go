package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// HashPassword provides a placeholder hashing strategy.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("password cannot be empty")
	}
	checksum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(checksum[:]), nil
}

// CompareHashAndPassword compares a stored checksum with plaintext input.
func CompareHashAndPassword(hash, plain string) error {
	expected, err := HashPassword(plain)
	if err != nil {
		return err
	}
	if expected != hash {
		return errors.New("password mismatch")
	}
	return nil
}
