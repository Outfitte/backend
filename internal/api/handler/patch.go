package handler

import (
	"encoding/json"

	"github.com/outfitte/backend/internal/domain"
)

// decodePatchNullable unmarshals a raw JSON value into a Nullable[T] field.
// Only call this when the key was present in the decoded map.
func decodePatchNullable[T any](raw json.RawMessage, dest *domain.Nullable[T]) error {
	var v *T
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	*dest = &v
	return nil
}
