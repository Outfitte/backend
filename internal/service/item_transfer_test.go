package service

import (
	"context"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type mockTransferRepo struct {
	transfers     []domain.ItemTransfer
	getErr        error
	saveErr       error
	deleteErr     error
	listErr       error
	hasPendingErr error
	hasPending    bool
	findPending   *domain.ItemTransfer
	findPendingErr error
}

func (m *mockTransferRepo) Get(_ context.Context, id string) (domain.ItemTransfer, error) {
	if m.getErr != nil {
		return domain.ItemTransfer{}, m.getErr
	}
	for _, t := range m.transfers {
		if t.GetID() == id {
			return t, nil
		}
	}
	return domain.ItemTransfer{}, domain.ErrNotFound
}

func (m *mockTransferRepo) Save(_ context.Context, t domain.ItemTransfer) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.transfers {
		if existing.GetID() == t.GetID() {
			m.transfers[i] = t
			return nil
		}
	}
	m.transfers = append(m.transfers, t)
	return nil
}

func (m *mockTransferRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, t := range m.transfers {
		if t.GetID() == id {
			m.transfers = append(m.transfers[:i], m.transfers[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockTransferRepo) ListBySender(_ context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.transfers, nil
}

func (m *mockTransferRepo) ListByRecipient(_ context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.transfers, nil
}

func (m *mockTransferRepo) FindPendingByItem(_ context.Context, _ string) (*domain.ItemTransfer, error) {
	return m.findPending, m.findPendingErr
}

func (m *mockTransferRepo) HasPending(_ context.Context, _ string) (bool, error) {
	return m.hasPending, m.hasPendingErr
}

type mockTransferTransactor struct {
	acceptResult domain.ItemTransfer
	acceptErr    error
}

func (m *mockTransferTransactor) Accept(_ context.Context, _ string) (domain.ItemTransfer, error) {
	return m.acceptResult, m.acceptErr
}

// helpers for building service with deterministic time/id

func newTestTransferService(
	repo *mockTransferRepo,
	transactor *mockTransferTransactor,
	itemRepo *mockItemRepo,
	userRepo *mockUserStore,
) *ItemTransferService {
	svc := NewItemTransferService(repo, transactor, itemRepo, userRepo)
	svc.idGen = func() string { return "transfer-1" }
	svc.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	return svc
}

// ── helpers for building test fixtures ────────────────────────────────────────

func makeItem(id, ownerID string) domain.Item {
	var item domain.Item
	item.ID = id
	item.OwnerID = ownerID
	return item
}

func makeArchivedItem(id, ownerID string) domain.Item {
	item := makeItem(id, ownerID)
	now := time.Now().UTC()
	item.ArchivedAt = &now
	return item
}

func makeDisposedItem(id, ownerID string) domain.Item {
	item := makeArchivedItem(id, ownerID)
	reason := domain.DisposalDonated
	item.DisposalReason = &reason
	return item
}

func makeUser(id, email string) domain.User {
	var u domain.User
	u.ID = id
	u.Email = email
	return u
}

func makePendingTransfer(id, itemID, senderID, recipientID string) domain.ItemTransfer {
	var t domain.ItemTransfer
	t.ID = id
	t.ItemID = itemID
	t.SenderID = senderID
	t.RecipientID = recipientID
	t.Status = domain.TransferStatusPending
	t.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return t
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestItemTransferServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferServiceCreateShouldReturnErrValidationWhenItemIDIsEmpty(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferServiceCreateShouldReturnErrValidationWhenRecipientIDIsEmpty(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: ""})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferServiceCreateShouldReturnErrSelfTransferWhenRecipientIsSender(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "sender-1"})
	require.ErrorIs(t, err, domain.ErrSelfTransfer)
}

func TestItemTransferServiceCreateShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceCreateShouldReturnErrForbiddenWhenCallerDoesNotOwnItem(t *testing.T) {
	item := makeItem("item-1", "other-owner")
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemTransferServiceCreateShouldReturnErrValidationWhenItemIsArchived(t *testing.T) {
	item := makeArchivedItem("item-1", "sender-1")
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferServiceCreateShouldReturnErrValidationWhenItemIsDisposed(t *testing.T) {
	item := makeDisposedItem("item-1", "sender-1")
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferServiceCreateShouldReturnErrNotFoundWhenRecipientDoesNotExist(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceCreateShouldReturnErrConflictWhenItemHasPendingTransfer(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	recipient := makeUser("recipient-1", "r@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{hasPending: true},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{recipient}},
	)

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestItemTransferServiceCreateShouldReturnErrorWhenHasPendingFails(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	recipient := makeUser("recipient-1", "r@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{hasPendingErr: domain.ErrIO},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{recipient}},
	)

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceCreateShouldReturnErrorWhenSaveFails(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	recipient := makeUser("recipient-1", "r@example.com")
	sender := makeUser("sender-1", "s@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{saveErr: domain.ErrIO},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceCreateShouldReturnTransferViewWhenInputIsValid(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	repo := &mockTransferRepo{}
	svc := newTestTransferService(repo, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{users: []domain.User{sender, recipient}})

	got, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1", TransferHistory: true})
	require.NoError(t, err)
	require.Equal(t, "transfer-1", got.Transfer.GetID())
	require.Equal(t, "item-1", got.Transfer.ItemID)
	require.Equal(t, "sender-1", got.Transfer.SenderID)
	require.Equal(t, "recipient-1", got.Transfer.RecipientID)
	require.Equal(t, domain.TransferStatusPending, got.Transfer.Status)
	require.True(t, got.Transfer.TransferHistory)
	require.Nil(t, got.Transfer.DecidedAt)
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), got.Transfer.CreatedAt)
	require.Equal(t, item, got.Item)
	require.Equal(t, "sender-1", got.Sender.ID)
	require.Equal(t, "sender@example.com", got.Sender.Email)
	require.Equal(t, "recipient-1", got.Recipient.ID)
	require.Equal(t, "recipient@example.com", got.Recipient.Email)
	require.Len(t, repo.transfers, 1)
}

func TestItemTransferServiceCreateShouldReturnErrorWhenSenderHydrationFails(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	recipient := makeUser("recipient-1", "r@example.com")
	// sender-1 not in user store → hydrate sender lookup fails
	repo := &mockTransferRepo{}
	svc := newTestTransferService(repo, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{users: []domain.User{recipient}})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceCreateShouldReturnErrorWhenRecipientHydrationFails(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "s@example.com")
	// recipient-1 not in user store → hydrate recipient lookup fails
	repo := &mockTransferRepo{}
	svc := newTestTransferService(repo, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{users: []domain.User{sender}})

	_, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// ── NewItemTransferService defaults ──────────────────────────────────────────

func TestNewItemTransferServiceShouldUseUUIDAndTimeDefaultsWhenNoOverrideProvided(t *testing.T) {
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	repo := &mockTransferRepo{}
	svc := NewItemTransferService(repo, &mockTransferTransactor{}, &mockItemRepo{items: []domain.Item{item}}, &mockUserStore{users: []domain.User{sender, recipient}})

	got, err := svc.Create(t.Context(), "sender-1", CreateTransferInput{ItemID: "item-1", RecipientID: "recipient-1"})
	require.NoError(t, err)
	require.NotEmpty(t, got.Transfer.GetID())
	require.False(t, got.Transfer.CreatedAt.IsZero())
}

// ── ListOutgoing ──────────────────────────────────────────────────────────────

func TestItemTransferServiceListOutgoingShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListOutgoing(ctx, "sender-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferServiceListOutgoingShouldReturnErrorWhenRepoFails(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{listErr: domain.ErrIO}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.ListOutgoing(t.Context(), "sender-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceListOutgoingShouldReturnErrorWhenHydrationFails(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{getErr: domain.ErrIO},
		&mockUserStore{},
	)

	_, err := svc.ListOutgoing(t.Context(), "sender-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceListOutgoingShouldReturnTransferViewsWhenSuccessful(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	got, err := svc.ListOutgoing(t.Context(), "sender-1", nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, tr, got[0].Transfer)
	require.Equal(t, item, got[0].Item)
	require.Equal(t, "sender-1", got[0].Sender.ID)
	require.Equal(t, "recipient-1", got[0].Recipient.ID)
}

// ── ListIncoming ──────────────────────────────────────────────────────────────

func TestItemTransferServiceListIncomingShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListIncoming(ctx, "recipient-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferServiceListIncomingShouldReturnErrorWhenRepoFails(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{listErr: domain.ErrIO}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.ListIncoming(t.Context(), "recipient-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceListIncomingShouldReturnErrorWhenHydrationFails(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{getErr: domain.ErrIO},
		&mockUserStore{},
	)

	_, err := svc.ListIncoming(t.Context(), "recipient-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceListIncomingShouldReturnTransferViewsWhenSuccessful(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	got, err := svc.ListIncoming(t.Context(), "recipient-1", nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, tr, got[0].Transfer)
	require.Equal(t, item, got[0].Item)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestItemTransferServiceGetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Get(ctx, "sender-1", "transfer-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferServiceGetShouldReturnErrNotFoundWhenTransferDoesNotExist(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Get(t.Context(), "sender-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceGetShouldReturnErrForbiddenWhenCallerIsNotParticipant(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Get(t.Context(), "outsider-99", "transfer-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemTransferServiceGetShouldReturnTransferViewWhenCallerIsSender(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	got, err := svc.Get(t.Context(), "sender-1", "transfer-1")
	require.NoError(t, err)
	require.Equal(t, tr, got.Transfer)
	require.Equal(t, item, got.Item)
}

func TestItemTransferServiceGetShouldReturnErrorWhenRecipientHydrationFails(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	// recipient-1 absent from store → hydrate fails at recipient lookup
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender}},
	)

	_, err := svc.Get(t.Context(), "sender-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceGetShouldReturnTransferViewWhenCallerIsRecipient(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	got, err := svc.Get(t.Context(), "recipient-1", "transfer-1")
	require.NoError(t, err)
	require.Equal(t, tr, got.Transfer)
}

// ── Accept ────────────────────────────────────────────────────────────────────

func TestItemTransferServiceAcceptShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Accept(ctx, "recipient-1", "transfer-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferServiceAcceptShouldReturnErrNotFoundWhenTransferDoesNotExist(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Accept(t.Context(), "recipient-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceAcceptShouldReturnErrForbiddenWhenCallerIsNotRecipient(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Accept(t.Context(), "sender-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemTransferServiceAcceptShouldReturnErrValidationWhenTransferIsNotPending(t *testing.T) {
	var tr domain.ItemTransfer
	tr.ID = "transfer-1"
	tr.ItemID = "item-1"
	tr.SenderID = "sender-1"
	tr.RecipientID = "recipient-1"
	tr.Status = domain.TransferStatusRejected
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Accept(t.Context(), "recipient-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferServiceAcceptShouldReturnErrorWhenTransactorFails(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{acceptErr: domain.ErrIO},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Accept(t.Context(), "recipient-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceAcceptShouldReturnTransferViewWhenSuccessful(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	decided := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	accepted := tr
	accepted.Status = domain.TransferStatusAccepted
	accepted.DecidedAt = &decided

	item := makeItem("item-1", "recipient-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{acceptResult: accepted},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	got, err := svc.Accept(t.Context(), "recipient-1", "transfer-1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusAccepted, got.Transfer.Status)
	require.NotNil(t, got.Transfer.DecidedAt)
}

// ── Reject ────────────────────────────────────────────────────────────────────

func TestItemTransferServiceRejectShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Reject(ctx, "recipient-1", "transfer-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferServiceRejectShouldReturnErrNotFoundWhenTransferDoesNotExist(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Reject(t.Context(), "recipient-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceRejectShouldReturnErrForbiddenWhenCallerIsNotRecipient(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Reject(t.Context(), "sender-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemTransferServiceRejectShouldReturnErrValidationWhenTransferIsNotPending(t *testing.T) {
	var tr domain.ItemTransfer
	tr.ID = "transfer-1"
	tr.ItemID = "item-1"
	tr.SenderID = "sender-1"
	tr.RecipientID = "recipient-1"
	tr.Status = domain.TransferStatusCancelled
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Reject(t.Context(), "recipient-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferServiceRejectShouldReturnErrorWhenSaveFails(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}, saveErr: domain.ErrIO},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Reject(t.Context(), "recipient-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceRejectShouldReturnTransferViewWhenSuccessful(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	repo := &mockTransferRepo{transfers: []domain.ItemTransfer{tr}}
	svc := newTestTransferService(
		repo,
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	got, err := svc.Reject(t.Context(), "recipient-1", "transfer-1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusRejected, got.Transfer.Status)
	require.NotNil(t, got.Transfer.DecidedAt)
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), *got.Transfer.DecidedAt)
	require.Equal(t, domain.TransferStatusRejected, repo.transfers[0].Status)
}

// ── fetchAndAuthorize ─────────────────────────────────────────────────────────

func TestFetchAndAuthorizeShouldPanicWhenRoleIsUnknown(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	require.Panics(t, func() {
		_, _ = svc.fetchAndAuthorize(t.Context(), "transfer-1", "sender-1", participantRole(99))
	})
}

// ── Cancel ────────────────────────────────────────────────────────────────────

func TestItemTransferServiceCancelShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Cancel(ctx, "sender-1", "transfer-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferServiceCancelShouldReturnErrNotFoundWhenTransferDoesNotExist(t *testing.T) {
	svc := newTestTransferService(&mockTransferRepo{}, &mockTransferTransactor{}, &mockItemRepo{}, &mockUserStore{})

	_, err := svc.Cancel(t.Context(), "sender-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferServiceCancelShouldReturnErrForbiddenWhenCallerIsNotSender(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Cancel(t.Context(), "recipient-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemTransferServiceCancelShouldReturnErrValidationWhenTransferIsNotPending(t *testing.T) {
	var tr domain.ItemTransfer
	tr.ID = "transfer-1"
	tr.ItemID = "item-1"
	tr.SenderID = "sender-1"
	tr.RecipientID = "recipient-1"
	tr.Status = domain.TransferStatusAccepted
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Cancel(t.Context(), "sender-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferServiceCancelShouldReturnErrorWhenSaveFails(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	svc := newTestTransferService(
		&mockTransferRepo{transfers: []domain.ItemTransfer{tr}, saveErr: domain.ErrIO},
		&mockTransferTransactor{},
		&mockItemRepo{},
		&mockUserStore{},
	)

	_, err := svc.Cancel(t.Context(), "sender-1", "transfer-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferServiceCancelShouldReturnTransferViewWhenSuccessful(t *testing.T) {
	tr := makePendingTransfer("transfer-1", "item-1", "sender-1", "recipient-1")
	item := makeItem("item-1", "sender-1")
	sender := makeUser("sender-1", "sender@example.com")
	recipient := makeUser("recipient-1", "recipient@example.com")
	repo := &mockTransferRepo{transfers: []domain.ItemTransfer{tr}}
	svc := newTestTransferService(
		repo,
		&mockTransferTransactor{},
		&mockItemRepo{items: []domain.Item{item}},
		&mockUserStore{users: []domain.User{sender, recipient}},
	)

	got, err := svc.Cancel(t.Context(), "sender-1", "transfer-1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusCancelled, got.Transfer.Status)
	require.NotNil(t, got.Transfer.DecidedAt)
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), *got.Transfer.DecidedAt)
	require.Equal(t, domain.TransferStatusCancelled, repo.transfers[0].Status)
}
