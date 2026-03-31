package domain_test

import (
	"testing"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// ── ValidatePurchasePrice ─────────────────────────────────────────────────────

func TestValidatePurchasePriceShouldReturnErrValidationWhenPriceIsEmpty(t *testing.T) {
	err := domain.ValidatePurchasePrice("")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchasePriceShouldReturnErrValidationWhenPriceIsNonNumeric(t *testing.T) {
	err := domain.ValidatePurchasePrice("abc")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchasePriceShouldReturnErrValidationWhenPriceIsNegative(t *testing.T) {
	err := domain.ValidatePurchasePrice("-1.50")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchasePriceShouldReturnNilWhenPriceIsZero(t *testing.T) {
	err := domain.ValidatePurchasePrice("0.00")
	require.NoError(t, err)
}

func TestValidatePurchasePriceShouldReturnNilWhenPriceIsValidDecimal(t *testing.T) {
	err := domain.ValidatePurchasePrice("123.45")
	require.NoError(t, err)
}

func TestValidatePurchasePriceShouldReturnNilWhenPriceIsWholeNumber(t *testing.T) {
	err := domain.ValidatePurchasePrice("10")
	require.NoError(t, err)
}

// ── ValidatePurchaseCurrency ──────────────────────────────────────────────────

func TestValidatePurchaseCurrencyShouldReturnErrValidationWhenCurrencyIsEmpty(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchaseCurrencyShouldReturnErrValidationWhenCurrencyIsTooShort(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("US")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchaseCurrencyShouldReturnErrValidationWhenCurrencyIsTooLong(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("USDD")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchaseCurrencyShouldReturnErrValidationWhenCurrencyContainsDigits(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("U5D")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchaseCurrencyShouldReturnErrValidationWhenCurrencyContainsSymbol(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("U$D")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchaseCurrencyShouldReturnNilWhenCurrencyIsUppercase(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("USD")
	require.NoError(t, err)
}

func TestValidatePurchaseCurrencyShouldReturnNilWhenCurrencyIsMixedCase(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("uSd")
	require.NoError(t, err)
}

func TestValidatePurchaseCurrencyShouldReturnNilWhenCurrencyIsLowercase(t *testing.T) {
	err := domain.ValidatePurchaseCurrency("eur")
	require.NoError(t, err)
}

// ── ValidatePurchaseDate ──────────────────────────────────────────────────────

func TestValidatePurchaseDateShouldReturnErrFutureDateNotAllowedWhenDateIsInTheFuture(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	err := domain.ValidatePurchaseDate(future)
	require.ErrorIs(t, err, domain.ErrFutureDateNotAllowed)
}

func TestValidatePurchaseDateShouldReturnNilWhenDateIsInThePast(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	err := domain.ValidatePurchaseDate(past)
	require.NoError(t, err)
}

// ── ValidatePurchasePair ──────────────────────────────────────────────────────

func TestValidatePurchasePairShouldReturnErrValidationWhenPriceIsSetAndCurrencyIsNil(t *testing.T) {
	price := "10.00"
	err := domain.ValidatePurchasePair(&price, nil)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchasePairShouldReturnErrValidationWhenCurrencyIsSetAndPriceIsNil(t *testing.T) {
	currency := "USD"
	err := domain.ValidatePurchasePair(nil, &currency)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidatePurchasePairShouldReturnNilWhenBothAreNil(t *testing.T) {
	err := domain.ValidatePurchasePair(nil, nil)
	require.NoError(t, err)
}

func TestValidatePurchasePairShouldReturnNilWhenBothAreSet(t *testing.T) {
	price := "10.00"
	currency := "USD"
	err := domain.ValidatePurchasePair(&price, &currency)
	require.NoError(t, err)
}
