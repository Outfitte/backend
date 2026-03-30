package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ValidatePurchasePrice validates that price is a non-negative decimal string.
// Returns ErrValidation if the string is empty, non-numeric, or represents a negative value.
func ValidatePurchasePrice(price string) error {
	if price == "" {
		return fmt.Errorf("%w: purchase price must not be empty", ErrValidation)
	}
	v, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return fmt.Errorf("%w: purchase price must be a valid decimal number", ErrValidation)
	}
	if v < 0 {
		return fmt.Errorf("%w: purchase price must be non-negative", ErrValidation)
	}
	return nil
}

// ValidatePurchaseCurrency validates that currency is exactly 3 ASCII letters.
// The input is normalised to uppercase before validation.
// Returns ErrValidation on failure.
func ValidatePurchaseCurrency(currency string) error {
	upper := strings.ToUpper(currency)
	if len(upper) != 3 {
		return fmt.Errorf("%w: purchase currency must be exactly 3 letters", ErrValidation)
	}
	for _, r := range upper {
		if r < 'A' || r > 'Z' {
			return fmt.Errorf("%w: purchase currency must contain only ASCII letters", ErrValidation)
		}
	}
	return nil
}

// ValidatePurchaseDate validates that date is not in the future.
// Returns ErrFutureDateNotAllowed if the date is after the current time.
func ValidatePurchaseDate(date time.Time) error {
	if date.After(time.Now()) {
		return ErrFutureDateNotAllowed
	}
	return nil
}

// ValidatePurchasePair validates that price and currency are either both set or both unset.
// Returns ErrValidation if exactly one of the two is non-nil.
func ValidatePurchasePair(price *string, currency *string) error {
	if (price != nil) != (currency != nil) {
		return fmt.Errorf("%w: purchase price and currency must both be set or both be absent", ErrValidation)
	}
	return nil
}
