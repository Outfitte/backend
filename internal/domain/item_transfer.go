package domain

import "time"

// TransferStatus is the lifecycle state of an ItemTransfer.
type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "pending"
	TransferStatusAccepted  TransferStatus = "accepted"
	TransferStatusRejected  TransferStatus = "rejected"
	TransferStatusCancelled TransferStatus = "cancelled"
)

// ItemTransfer represents a pending or resolved transfer of an item between users.
type ItemTransfer struct {
	uniqueEntity
	ItemID          string
	SenderID        string
	RecipientID     string
	Status          TransferStatus
	TransferHistory bool // whether the item's wear history is included in the transfer
	CreatedAt       time.Time
	DecidedAt       *time.Time // non-nil once status transitions away from pending
}
