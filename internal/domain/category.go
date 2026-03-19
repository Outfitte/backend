package domain

// CategoryUncategorised is the sentinel value for an uncategorised item.
// A category_id of nil on Item means uncategorised; no category record is stored for this state.
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
