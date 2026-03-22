package sqlstore

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/domain"
)

func TestOpenSQLiteShouldReturnErrIOWhenOpenFails(t *testing.T) {
	old := sqlOpenFn
	sqlOpenFn = func(_, _ string) (*sql.DB, error) {
		return nil, errors.New("injected open failure")
	}
	t.Cleanup(func() { sqlOpenFn = old })

	_, err := openSQLite(t.Context(), ":memory:")
	require.ErrorIs(t, err, domain.ErrIO)
}
