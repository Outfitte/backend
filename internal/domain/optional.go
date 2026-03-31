package domain

// Nullable[T] represents a PATCH input field with three states:
//
//	nil              — field was absent from the request body; preserve existing value
//	pointer to nil   — field was explicitly set to null; clear the value
//	pointer to value — field was set to a new value; update to this value
type Nullable[T any] = **T
