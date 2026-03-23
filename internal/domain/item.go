package domain

import "time"

// DisposalReason represents the reason an item was disposed of.
type DisposalReason string

const (
	DisposalDonated   DisposalReason = "donated"
	DisposalSold      DisposalReason = "sold"
	DisposalDiscarded DisposalReason = "discarded"
	DisposalLost      DisposalReason = "lost"
	DisposalOther     DisposalReason = "other"
)

type Item struct {
	uniqueEntity
	OwnerID        string
	Name           string
	Brand          *string // optional
	CategoryID     *string // optional; nil = uncategorised
	Color          *string // optional
	Metadata       ItemMetadata
	Photos         []ItemPhoto
	LocationID     *string    // optional
	PurchasePrice  *string    // optional, deferred to M4+; string to avoid decimal dep
	PurchaseDate   *time.Time // optional, deferred to M4+
	CreatedAt      time.Time
	ArchivedAt     *time.Time      // non-nil means item is archived
	DisposalReason *DisposalReason // non-nil means item is disposed (implies archived)
}
