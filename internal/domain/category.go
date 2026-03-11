package domain

type Category struct {
	uniqueEntity
	Label    string
	IsPreset bool
}
