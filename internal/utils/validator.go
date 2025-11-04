package utils

import "errors"

// ValidateStruct is a placeholder for request validation logic.
func ValidateStruct(v interface{}) error {
	if v == nil {
		return errors.New("validation target cannot be nil")
	}
	return nil
}
