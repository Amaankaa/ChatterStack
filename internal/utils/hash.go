package utils

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const passwordMinLength = 8

// HashPassword applies bcrypt hashing with the default cost.
func HashPassword(plain string) (string, error) {
	if len(plain) < passwordMinLength {
		return "", errors.New("password must be at least 8 characters long")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// CompareHashAndPassword verifies a plaintext password against a stored hash.
func CompareHashAndPassword(hash, plain string) error {
	if hash == "" {
		return errors.New("stored password hash cannot be empty")
	}
	if len(plain) < passwordMinLength {
		return errors.New("password must be at least 8 characters long")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
