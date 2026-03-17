package domain

import (
	"fmt"
	"strings"
	"unicode"
)

// ItemMetadata holds dynamic key-value pairs for an item, including
// category hint fields and user-defined custom fields.
// Serialised as a JSON column in the DB and as a nested object in the JSON file store.
type ItemMetadata struct {
	Fields map[string]string
}

const (
	metadataMaxKeys       = 50
	metadataMaxKeyLen     = 64
	metadataMaxValueLen   = 512
)

// ValidateMetadataKey validates a single metadata key against the rules:
// max 64 characters, only alphanumeric characters and spaces, no leading/trailing spaces.
func ValidateMetadataKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: metadata key must not be empty", ErrValidation)
	}
	if len(key) > metadataMaxKeyLen {
		return fmt.Errorf("%w: metadata key exceeds maximum length of %d characters", ErrValidation, metadataMaxKeyLen)
	}
	if strings.TrimSpace(key) != key {
		return fmt.Errorf("%w: metadata key must not have leading or trailing spaces", ErrValidation)
	}
	for _, r := range key {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' {
			return fmt.Errorf("%w: metadata key contains invalid character %q (only alphanumeric and spaces allowed)", ErrValidation, r)
		}
	}
	return nil
}

// ValidateMetadata validates all keys, values, and field count of an ItemMetadata.
// Rules: max 50 fields, each key validated by ValidateMetadataKey,
// each value max 512 characters.
func ValidateMetadata(m ItemMetadata) error {
	if len(m.Fields) > metadataMaxKeys {
		return fmt.Errorf("%w: metadata exceeds maximum of %d fields", ErrValidation, metadataMaxKeys)
	}
	for k, v := range m.Fields {
		if err := ValidateMetadataKey(k); err != nil {
			return err
		}
		if len(v) > metadataMaxValueLen {
			return fmt.Errorf("%w: metadata value for key %q exceeds maximum length of %d characters", ErrValidation, k, metadataMaxValueLen)
		}
	}
	return nil
}
