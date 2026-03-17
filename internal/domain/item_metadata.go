package domain

// ItemMetadata holds dynamic key-value pairs for an item, including
// category hint fields and user-defined custom fields.
// Serialised as a JSON column in the DB and as a nested object in the JSON file store.
type ItemMetadata struct {
	Fields map[string]string
}
