package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// CreateTransferInput holds the fields required to initiate an item transfer.
type CreateTransferInput struct {
	ItemID          string
	RecipientID     string
	TransferHistory bool
}

// TransferView is the hydrated transfer returned to callers.
type TransferView struct {
	Transfer  domain.ItemTransfer
	Item      domain.Item
	Sender    UserSummary
	Recipient UserSummary
}

// ItemTransferService orchestrates the item transfer lifecycle.
type ItemTransferService struct {
	transfers  ports.ItemTransferRepository
	transactor ports.ItemTransferTransactor
	items      ports.ItemRepository
	users      ports.UserRepository
	idGen      func() string
	now        func() time.Time
	log        *slog.Logger
}

// NewItemTransferService constructs an ItemTransferService backed by the given ports.
func NewItemTransferService(
	transfers ports.ItemTransferRepository,
	transactor ports.ItemTransferTransactor,
	items ports.ItemRepository,
	users ports.UserRepository,
	log *slog.Logger,
) *ItemTransferService {
	return &ItemTransferService{
		transfers:  transfers,
		transactor: transactor,
		items:      items,
		users:      users,
		idGen:      uuid.NewString,
		now:        func() time.Time { return time.Now().UTC() },
		log:        log,
	}
}

// Create initiates a new item transfer from callerID to the specified recipient.
func (s *ItemTransferService) Create(ctx context.Context, callerID string, input CreateTransferInput) (TransferView, error) {
	if err := ctx.Err(); err != nil {
		return TransferView{}, err
	}
	if input.ItemID == "" || input.RecipientID == "" {
		return TransferView{}, domain.ErrValidation
	}
	if input.RecipientID == callerID {
		return TransferView{}, domain.ErrSelfTransfer
	}
	item, err := s.items.Get(ctx, input.ItemID)
	if err != nil {
		return TransferView{}, err
	}
	if item.OwnerID != callerID {
		return TransferView{}, domain.ErrForbidden
	}
	if item.ArchivedAt != nil {
		return TransferView{}, domain.ErrValidation
	}
	if _, err := s.users.Get(ctx, input.RecipientID); err != nil {
		return TransferView{}, err
	}
	pending, err := s.transfers.HasPending(ctx, input.ItemID)
	if err != nil {
		return TransferView{}, err
	}
	if pending {
		return TransferView{}, domain.ErrConflict
	}
	transfer, err := s.saveNewTransfer(ctx, callerID, input)
	if err != nil {
		return TransferView{}, err
	}
	return s.hydrate(ctx, transfer)
}

func (s *ItemTransferService) saveNewTransfer(ctx context.Context, senderID string, input CreateTransferInput) (domain.ItemTransfer, error) {
	var t domain.ItemTransfer
	t.ID = s.idGen()
	t.ItemID = input.ItemID
	t.SenderID = senderID
	t.RecipientID = input.RecipientID
	t.Status = domain.TransferStatusPending
	t.TransferHistory = input.TransferHistory
	t.CreatedAt = s.now()
	if err := s.transfers.Save(ctx, t); err != nil {
		return domain.ItemTransfer{}, err
	}
	return t, nil
}

func (s *ItemTransferService) hydrate(ctx context.Context, transfer domain.ItemTransfer) (TransferView, error) {
	item, err := s.items.Get(ctx, transfer.ItemID)
	if err != nil {
		return TransferView{}, err
	}
	sender, err := s.users.Get(ctx, transfer.SenderID)
	if err != nil {
		return TransferView{}, err
	}
	recipient, err := s.users.Get(ctx, transfer.RecipientID)
	if err != nil {
		return TransferView{}, err
	}
	return TransferView{
		Transfer:  transfer,
		Item:      item,
		Sender:    UserSummary{ID: sender.GetID(), Email: sender.Email},
		Recipient: UserSummary{ID: recipient.GetID(), Email: recipient.Email},
	}, nil
}

// ListOutgoing returns all transfers sent by callerID, optionally filtered by status.
func (s *ItemTransferService) ListOutgoing(ctx context.Context, callerID string, statusFilter *domain.TransferStatus) ([]TransferView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	transfers, err := s.transfers.ListBySender(ctx, callerID, statusFilter)
	if err != nil {
		return nil, err
	}
	return s.hydrateAll(ctx, transfers)
}

func (s *ItemTransferService) hydrateAll(ctx context.Context, transfers []domain.ItemTransfer) ([]TransferView, error) {
	views := make([]TransferView, 0, len(transfers))
	for _, t := range transfers {
		view, err := s.hydrate(ctx, t)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

// ListIncoming returns all transfers addressed to callerID, optionally filtered by status.
func (s *ItemTransferService) ListIncoming(ctx context.Context, callerID string, statusFilter *domain.TransferStatus) ([]TransferView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	transfers, err := s.transfers.ListByRecipient(ctx, callerID, statusFilter)
	if err != nil {
		return nil, err
	}
	return s.hydrateAll(ctx, transfers)
}

// Get returns the hydrated transfer identified by transferID if callerID is a participant.
func (s *ItemTransferService) Get(ctx context.Context, callerID, transferID string) (TransferView, error) {
	if err := ctx.Err(); err != nil {
		return TransferView{}, err
	}
	transfer, err := s.transfers.Get(ctx, transferID)
	if err != nil {
		return TransferView{}, err
	}
	if transfer.SenderID != callerID && transfer.RecipientID != callerID {
		return TransferView{}, domain.ErrForbidden
	}
	return s.hydrate(ctx, transfer)
}

// Accept delegates the acceptance of a pending transfer to the transactor.
func (s *ItemTransferService) Accept(ctx context.Context, callerID, transferID string) (TransferView, error) {
	if err := ctx.Err(); err != nil {
		return TransferView{}, err
	}
	transfer, err := s.transfers.Get(ctx, transferID)
	if err != nil {
		return TransferView{}, err
	}
	if transfer.RecipientID != callerID {
		return TransferView{}, domain.ErrForbidden
	}
	if transfer.Status != domain.TransferStatusPending {
		return TransferView{}, domain.ErrValidation
	}
	accepted, err := s.transactor.Accept(ctx, transferID)
	if err != nil {
		return TransferView{}, err
	}
	return s.hydrate(ctx, accepted)
}

// Reject marks a pending transfer as rejected by the recipient.
func (s *ItemTransferService) Reject(ctx context.Context, callerID, transferID string) (TransferView, error) {
	if err := ctx.Err(); err != nil {
		return TransferView{}, err
	}
	transfer, err := s.fetchAndAuthorize(ctx, transferID, callerID, recipientOnly)
	if err != nil {
		return TransferView{}, err
	}
	decided := s.now()
	transfer.Status = domain.TransferStatusRejected
	transfer.DecidedAt = &decided
	if err := s.transfers.Save(ctx, transfer); err != nil {
		return TransferView{}, err
	}
	return s.hydrate(ctx, transfer)
}

// Cancel marks a pending transfer as cancelled by the sender.
func (s *ItemTransferService) Cancel(ctx context.Context, callerID, transferID string) (TransferView, error) {
	if err := ctx.Err(); err != nil {
		return TransferView{}, err
	}
	transfer, err := s.fetchAndAuthorize(ctx, transferID, callerID, senderOnly)
	if err != nil {
		return TransferView{}, err
	}
	decided := s.now()
	transfer.Status = domain.TransferStatusCancelled
	transfer.DecidedAt = &decided
	if err := s.transfers.Save(ctx, transfer); err != nil {
		return TransferView{}, err
	}
	return s.hydrate(ctx, transfer)
}

type participantRole int

const (
	senderOnly    participantRole = iota
	recipientOnly participantRole = iota
)

func (s *ItemTransferService) fetchAndAuthorize(ctx context.Context, transferID, callerID string, role participantRole) (domain.ItemTransfer, error) {
	transfer, err := s.transfers.Get(ctx, transferID)
	if err != nil {
		return domain.ItemTransfer{}, err
	}
	switch role {
	case recipientOnly:
		if transfer.RecipientID != callerID {
			return domain.ItemTransfer{}, domain.ErrForbidden
		}
	case senderOnly:
		if transfer.SenderID != callerID {
			return domain.ItemTransfer{}, domain.ErrForbidden
		}
	}
	if transfer.Status != domain.TransferStatusPending {
		return domain.ItemTransfer{}, domain.ErrValidation
	}
	return transfer, nil
}
