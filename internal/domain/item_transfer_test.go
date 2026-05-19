package domain_test

import (
	"testing"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
	"github.com/stretchr/testify/assert"
)

func TestItemTransferShouldImplementPortsEntityWhenGetIDCalled(t *testing.T) {
	iface := (*ports.Entity)(nil)
	assert.Implements(t, iface, domain.ItemTransfer{})

	var tr domain.ItemTransfer
	tr.ID = "transfer-99"
	assert.Equal(t, "transfer-99", tr.GetID())
}

func TestItemTransferDecidedAtShouldBeNilWhenPending(t *testing.T) {
	var tr domain.ItemTransfer
	tr.ID = "transfer-1"
	tr.Status = domain.TransferStatusPending
	assert.Nil(t, tr.DecidedAt)
}

func TestTransferStatusConstantsShouldBeDistinctWhenCompared(t *testing.T) {
	statuses := []domain.TransferStatus{
		domain.TransferStatusPending,
		domain.TransferStatusAccepted,
		domain.TransferStatusRejected,
		domain.TransferStatusCancelled,
	}
	seen := make(map[domain.TransferStatus]bool)
	for _, s := range statuses {
		assert.False(t, seen[s], "duplicate TransferStatus constant: %s", s)
		seen[s] = true
	}
}

func TestTransferStatusConstantsShouldHaveExpectedValuesWhenDefined(t *testing.T) {
	assert.Equal(t, domain.TransferStatus("pending"), domain.TransferStatusPending)
	assert.Equal(t, domain.TransferStatus("accepted"), domain.TransferStatusAccepted)
	assert.Equal(t, domain.TransferStatus("rejected"), domain.TransferStatusRejected)
	assert.Equal(t, domain.TransferStatus("cancelled"), domain.TransferStatusCancelled)
}

func TestItemTransferShouldHoldAllFieldsWhenConstructed(t *testing.T) {
	now := time.Now()
	decided := now.Add(time.Hour)

	var tr domain.ItemTransfer
	tr.ID = "transfer-1"
	tr.ItemID = "item-1"
	tr.SenderID = "sender-1"
	tr.RecipientID = "recipient-1"
	tr.Status = domain.TransferStatusAccepted
	tr.TransferHistory = true
	tr.CreatedAt = now
	tr.DecidedAt = &decided

	assert.Equal(t, "transfer-1", tr.ID)
	assert.Equal(t, "item-1", tr.ItemID)
	assert.Equal(t, "sender-1", tr.SenderID)
	assert.Equal(t, "recipient-1", tr.RecipientID)
	assert.Equal(t, domain.TransferStatusAccepted, tr.Status)
	assert.True(t, tr.TransferHistory)
	assert.Equal(t, now, tr.CreatedAt)
	assert.Equal(t, &decided, tr.DecidedAt)
}
