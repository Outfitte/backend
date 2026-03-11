package domain

// uniqueEntity holds the string UUID shared by all domain entities
// and satisfies the ports.Entity interface via GetID.
type uniqueEntity struct {
	ID string
}

func (e uniqueEntity) GetID() string { return e.ID }
