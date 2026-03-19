package domain

// CategoryUncategorised is the sentinel string value representing the uncategorised state.
// On Item, CategoryID is a *string where nil means uncategorised — no category record is stored.
// This constant is provided for use in string-based contexts (e.g. API serialisation) where
// an explicit empty value must be distinguished from a missing field.
const CategoryUncategorised = ""

// FieldHint provides UI guidance for a structured attribute of a category.
type FieldHint struct {
	Key         string
	Label       string
	Placeholder string
}

type Category struct {
	uniqueEntity
	Label      string
	IsPreset   bool
	FieldHints []FieldHint
}
