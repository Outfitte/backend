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
	LocationID       *string    // optional
	PurchasePrice    *string    // optional; string to avoid decimal dep
	// PurchaseCurrency must always accompany PurchasePrice — if one is set, both must be
	// set. This invariant is enforced by the service layer, not the domain struct.
	PurchaseCurrency *string    // optional; 3-letter ISO 4217 code (e.g. "USD", "PLN", "EUR")
	PurchaseDate     *time.Time // optional
	SellerURL        *string    // optional; single URL per item, non-empty when provided
	CreatedAt        time.Time
	ArchivedAt       *time.Time      // non-nil means item is archived
	DisposalReason   *DisposalReason // non-nil means item is disposed (implies archived)
}
