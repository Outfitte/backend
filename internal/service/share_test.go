package service

import (
	"context"
	"testing"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
	"github.com/stretchr/testify/require"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

// mockShareRepo is an in-memory ports.ShareRepository for tests.
type mockShareRepo struct {
	shares              []domain.Share
	getErr              error
	saveErr             error
	deleteErr           error
	listByOwnerErr      error
	listByRecipientErr  error
	findByTargetErr     error
	findByTargetResult  *domain.Share
	deleteByTargetErr   error
	hasDirectAccessErr  error
	hasDirectAccessResult bool
	listByRecipientAndTypeErr    error
	listByRecipientAndTypeResult []domain.Share
}

func (m *mockShareRepo) Get(_ context.Context, id string) (domain.Share, error) {
	if m.getErr != nil {
		return domain.Share{}, m.getErr
	}
	for _, s := range m.shares {
		if s.GetID() == id {
			return s, nil
		}
	}
	return domain.Share{}, domain.ErrNotFound
}

func (m *mockShareRepo) Save(_ context.Context, share domain.Share) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.shares {
		if existing.GetID() == share.GetID() {
			m.shares[i] = share
			return nil
		}
	}
	m.shares = append(m.shares, share)
	return nil
}

func (m *mockShareRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, s := range m.shares {
		if s.GetID() == id {
			m.shares = append(m.shares[:i], m.shares[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockShareRepo) ListByOwner(_ context.Context, ownerID string) ([]domain.Share, error) {
	if m.listByOwnerErr != nil {
		return nil, m.listByOwnerErr
	}
	var result []domain.Share
	for _, s := range m.shares {
		if s.OwnerID == ownerID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockShareRepo) ListByRecipient(_ context.Context, recipientID string) ([]domain.Share, error) {
	if m.listByRecipientErr != nil {
		return nil, m.listByRecipientErr
	}
	var result []domain.Share
	for _, s := range m.shares {
		if s.RecipientID == recipientID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockShareRepo) FindByTarget(_ context.Context, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string) (*domain.Share, error) {
	if m.findByTargetErr != nil {
		return nil, m.findByTargetErr
	}
	if m.findByTargetResult != nil {
		return m.findByTargetResult, nil
	}
	for i, s := range m.shares {
		if s.OwnerID == ownerID && s.RecipientID == recipientID && s.TargetType == targetType && s.TargetID == targetID {
			return &m.shares[i], nil
		}
	}
	return nil, nil
}

func (m *mockShareRepo) DeleteByTarget(_ context.Context, _ domain.ShareTargetType, _ string) error {
	return m.deleteByTargetErr
}

func (m *mockShareRepo) HasDirectAccess(_ context.Context, _ string, _ domain.ShareTargetType, _ string) (bool, error) {
	if m.hasDirectAccessErr != nil {
		return false, m.hasDirectAccessErr
	}
	return m.hasDirectAccessResult, nil
}

func (m *mockShareRepo) ListByRecipientAndType(_ context.Context, _ string, _ domain.ShareTargetType) ([]domain.Share, error) {
	if m.listByRecipientAndTypeErr != nil {
		return nil, m.listByRecipientAndTypeErr
	}
	return m.listByRecipientAndTypeResult, nil
}

// helper: builds a ShareService with default empty mocks.
func newTestShareService(
	shareRepo ports.ShareRepository,
	userRepo ports.UserRepository,
	itemRepo ports.ItemRepository,
	outfitRepo ports.OutfitRepository,
	locationRepo ports.LocationRepository,
) *ShareService {
	return NewShareService(shareRepo, userRepo, itemRepo, outfitRepo, locationRepo)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestShareServiceCreateShouldReturnErrorWhenRepoSaveFails(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	shareRepo := &mockShareRepo{saveErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareServiceCreateShouldReturnErrNotFoundWhenRecipientDoesNotExist(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceCreateShouldReturnErrSelfShareWhenRecipientIsTheCaller(t *testing.T) {
	var user domain.User
	user.ID = "owner-1"
	user.Email = "owner@example.com"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{user}}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "owner-1",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrSelfShare)
}

func TestShareServiceCreateShouldReturnErrNotFoundWhenItemTargetDoesNotExist(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceCreateShouldReturnErrForbiddenWhenCallerDoesNotOwnItemTarget(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-2"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestShareServiceCreateShouldReturnErrNotFoundWhenOutfitTargetDoesNotExist(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetOutfit,
		TargetID:    "outfit-1",
	})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceCreateShouldReturnErrForbiddenWhenCallerDoesNotOwnOutfitTarget(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	var outfit domain.Outfit
	outfit.ID = "outfit-1"
	outfit.OwnerID = "owner-2"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{}, &mockOutfitRepo{outfits: []domain.Outfit{outfit}}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetOutfit,
		TargetID:    "outfit-1",
	})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestShareServiceCreateShouldReturnErrNotFoundWhenLocationTargetDoesNotExist(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetLocation,
		TargetID:    "loc-1",
	})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceCreateShouldReturnErrForbiddenWhenCallerDoesNotOwnLocationTarget(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc}})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetLocation,
		TargetID:    "loc-1",
	})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestShareServiceCreateShouldReturnErrDuplicateShareWhenShareAlreadyExists(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	var existing domain.Share
	existing.ID = "share-1"
	existing.OwnerID = "owner-1"
	existing.RecipientID = "user-2"
	existing.TargetType = domain.ShareTargetItem
	existing.TargetID = "item-1"

	shareRepo := &mockShareRepo{findByTargetResult: &existing}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrDuplicateShare)
}

func TestShareServiceCreateShouldReturnErrorWhenFindByTargetFails(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	shareRepo := &mockShareRepo{findByTargetErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceCreateShouldReturnShareWhenInputIsValid(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	shareRepo := &mockShareRepo{}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetItem,
		TargetID:    "item-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, got.GetID())
	require.Equal(t, "owner-1", got.OwnerID)
	require.Equal(t, "user-2", got.RecipientID)
	require.Equal(t, domain.ShareTargetItem, got.TargetType)
	require.Equal(t, "item-1", got.TargetID)
	require.False(t, got.CreatedAt.IsZero())
	require.Len(t, shareRepo.shares, 1)
}

// ── ListOutgoing ──────────────────────────────────────────────────────────────

func TestShareServiceListOutgoingShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListOutgoing(ctx, "owner-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareServiceListOutgoingShouldReturnErrorWhenRepoListByOwnerFails(t *testing.T) {
	shareRepo := &mockShareRepo{listByOwnerErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.ListOutgoing(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListOutgoingShouldReturnErrorWhenRecipientUserGetFails(t *testing.T) {
	var share domain.Share
	share.ID = "share-1"
	share.OwnerID = "owner-1"
	share.RecipientID = "user-2"

	shareRepo := &mockShareRepo{shares: []domain.Share{share}}
	userRepo := &mockUserStore{getByEmailErr: domain.ErrIO}
	// Override Get to fail
	userRepo.users = nil
	svc := newTestShareService(shareRepo, userRepo, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.ListOutgoing(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceListOutgoingShouldReturnShareViewsWithRecipientSummaries(t *testing.T) {
	var share domain.Share
	share.ID = "share-1"
	share.OwnerID = "owner-1"
	share.RecipientID = "user-2"
	share.TargetType = domain.ShareTargetItem
	share.TargetID = "item-1"

	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	shareRepo := &mockShareRepo{shares: []domain.Share{share}}
	userRepo := &mockUserStore{users: []domain.User{recipient}}
	svc := newTestShareService(shareRepo, userRepo, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.ListOutgoing(t.Context(), "owner-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, share, got[0].Share)
	require.Equal(t, "user-2", got[0].Recipient.ID)
	require.Equal(t, "user2@example.com", got[0].Recipient.Email)
}

// ── ListSharedWithMe ──────────────────────────────────────────────────────────

func TestShareServiceListSharedWithMeShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListSharedWithMe(ctx, "user-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenListByRecipientAndTypeForItemsFails(t *testing.T) {
	shareRepo := &mockShareRepo{listByRecipientAndTypeErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenItemGetFails(t *testing.T) {
	var itemShare domain.Share
	itemShare.ID = "share-1"
	itemShare.OwnerID = "owner-1"
	itemShare.RecipientID = "user-1"
	itemShare.TargetType = domain.ShareTargetItem
	itemShare.TargetID = "item-1"

	shareRepo := &mockShareRepoByType{
		itemShares: []domain.Share{itemShare},
	}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{getErr: domain.ErrIO}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenOwnerGetFailsForItem(t *testing.T) {
	var itemShare domain.Share
	itemShare.ID = "share-1"
	itemShare.OwnerID = "owner-1"
	itemShare.RecipientID = "user-1"
	itemShare.TargetType = domain.ShareTargetItem
	itemShare.TargetID = "item-1"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	shareRepo := &mockShareRepoByType{itemShares: []domain.Share{itemShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceListSharedWithMeShouldReturnSharedItemsWithOwnerSummaries(t *testing.T) {
	var itemShare domain.Share
	itemShare.ID = "share-1"
	itemShare.OwnerID = "owner-1"
	itemShare.RecipientID = "user-1"
	itemShare.TargetType = domain.ShareTargetItem
	itemShare.TargetID = "item-1"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	var owner domain.User
	owner.ID = "owner-1"
	owner.Email = "owner@example.com"

	shareRepo := &mockShareRepoByType{itemShares: []domain.Share{itemShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{owner}}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	require.Equal(t, item, got.Items[0].Entity)
	require.Equal(t, "owner-1", got.Items[0].SharedBy.ID)
	require.Equal(t, "owner@example.com", got.Items[0].SharedBy.Email)
	require.Empty(t, got.Outfits)
	require.Empty(t, got.Locations)
}

func TestShareServiceListSharedWithMeShouldReturnSharedOutfitsWithOwnerSummaries(t *testing.T) {
	var outfitShare domain.Share
	outfitShare.ID = "share-2"
	outfitShare.OwnerID = "owner-1"
	outfitShare.RecipientID = "user-1"
	outfitShare.TargetType = domain.ShareTargetOutfit
	outfitShare.TargetID = "outfit-1"

	var outfit domain.Outfit
	outfit.ID = "outfit-1"
	outfit.OwnerID = "owner-1"

	var owner domain.User
	owner.ID = "owner-1"
	owner.Email = "owner@example.com"

	shareRepo := &mockShareRepoByType{outfitShares: []domain.Share{outfitShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{owner}}, &mockItemRepo{}, &mockOutfitRepo{outfits: []domain.Outfit{outfit}}, &mockLocationRepo{})

	got, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.NoError(t, err)
	require.Len(t, got.Outfits, 1)
	require.Equal(t, outfit, got.Outfits[0].Entity)
	require.Equal(t, "owner-1", got.Outfits[0].SharedBy.ID)
	require.Empty(t, got.Items)
	require.Empty(t, got.Locations)
}

func TestShareServiceListSharedWithMeShouldReturnSharedLocationsWithItemsAndOwnerSummaries(t *testing.T) {
	var locShare domain.Share
	locShare.ID = "share-3"
	locShare.OwnerID = "owner-1"
	locShare.RecipientID = "user-1"
	locShare.TargetType = domain.ShareTargetLocation
	locShare.TargetID = "loc-1"

	locID := "loc-1"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.LocationID = &locID

	var owner domain.User
	owner.ID = "owner-1"
	owner.Email = "owner@example.com"

	shareRepo := &mockShareRepoByType{locationShares: []domain.Share{locShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{owner}}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc}})

	got, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.NoError(t, err)
	require.Len(t, got.Locations, 1)
	require.Equal(t, loc, got.Locations[0].Location)
	require.Equal(t, "owner-1", got.Locations[0].SharedBy.ID)
	require.Len(t, got.Locations[0].Items, 1)
	require.Equal(t, item, got.Locations[0].Items[0])
	require.Empty(t, got.Items)
	require.Empty(t, got.Outfits)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenHydrateSharedOutfitsFails(t *testing.T) {
	var outfitShare domain.Share
	outfitShare.ID = "share-2"
	outfitShare.OwnerID = "owner-1"
	outfitShare.RecipientID = "user-1"
	outfitShare.TargetType = domain.ShareTargetOutfit
	outfitShare.TargetID = "outfit-1"

	shareRepo := &mockShareRepoByType{outfitShares: []domain.Share{outfitShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{getErr: domain.ErrIO}, &mockLocationRepo{})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenHydrateSharedLocationsFails(t *testing.T) {
	var locShare domain.Share
	locShare.ID = "share-3"
	locShare.OwnerID = "owner-1"
	locShare.RecipientID = "user-1"
	locShare.TargetType = domain.ShareTargetLocation
	locShare.TargetID = "loc-1"

	shareRepo := &mockShareRepoByType{locationShares: []domain.Share{locShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{getErr: domain.ErrIO})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenOwnerGetFailsForOutfit(t *testing.T) {
	var outfitShare domain.Share
	outfitShare.ID = "share-2"
	outfitShare.OwnerID = "owner-1"
	outfitShare.RecipientID = "user-1"
	outfitShare.TargetType = domain.ShareTargetOutfit
	outfitShare.TargetID = "outfit-1"

	var outfit domain.Outfit
	outfit.ID = "outfit-1"
	outfit.OwnerID = "owner-1"

	shareRepo := &mockShareRepoByType{outfitShares: []domain.Share{outfitShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{outfits: []domain.Outfit{outfit}}, &mockLocationRepo{})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenOwnerGetFailsForLocation(t *testing.T) {
	var locShare domain.Share
	locShare.ID = "share-3"
	locShare.OwnerID = "owner-1"
	locShare.RecipientID = "user-1"
	locShare.TargetType = domain.ShareTargetLocation
	locShare.TargetID = "loc-1"

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	shareRepo := &mockShareRepoByType{locationShares: []domain.Share{locShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc}})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenCollectLocationItemsFails(t *testing.T) {
	var locShare domain.Share
	locShare.ID = "share-3"
	locShare.OwnerID = "owner-1"
	locShare.RecipientID = "user-1"
	locShare.TargetType = domain.ShareTargetLocation
	locShare.TargetID = "loc-1"

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var owner domain.User
	owner.ID = "owner-1"
	owner.Email = "owner@example.com"

	shareRepo := &mockShareRepoByType{locationShares: []domain.Share{locShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{owner}}, &mockItemRepo{listByOwnerErr: domain.ErrIO}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc}})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListSharedWithMeShouldIncludeDescendantLocationsItemsWhenLocationIsShared(t *testing.T) {
	var locShare domain.Share
	locShare.ID = "share-3"
	locShare.OwnerID = "owner-1"
	locShare.RecipientID = "user-1"
	locShare.TargetType = domain.ShareTargetLocation
	locShare.TargetID = "loc-root"

	rootID := "loc-root"
	childID := "loc-child"

	var root domain.Location
	root.ID = "loc-root"
	root.OwnerID = "owner-1"

	var child domain.Location
	child.ID = "loc-child"
	child.OwnerID = "owner-1"
	child.ParentID = &rootID

	var itemInRoot domain.Item
	itemInRoot.ID = "item-root"
	itemInRoot.OwnerID = "owner-1"
	itemInRoot.LocationID = &rootID

	var itemInChild domain.Item
	itemInChild.ID = "item-child"
	itemInChild.OwnerID = "owner-1"
	itemInChild.LocationID = &childID

	var owner domain.User
	owner.ID = "owner-1"
	owner.Email = "owner@example.com"

	shareRepo := &mockShareRepoByType{locationShares: []domain.Share{locShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{owner}}, &mockItemRepo{items: []domain.Item{itemInRoot, itemInChild}}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{root, child}})

	got, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.NoError(t, err)
	require.Len(t, got.Locations, 1)
	require.ElementsMatch(t, []domain.Item{itemInRoot, itemInChild}, got.Locations[0].Items)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenListByRecipientAndTypeForOutfitsFails(t *testing.T) {
	shareRepo := &mockShareRepoByType{outfitListErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenListByRecipientAndTypeForLocationsFails(t *testing.T) {
	shareRepo := &mockShareRepoByType{locationListErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceListSharedWithMeShouldReturnErrorWhenLocationListByOwnerFails(t *testing.T) {
	var locShare domain.Share
	locShare.ID = "share-3"
	locShare.OwnerID = "owner-1"
	locShare.RecipientID = "user-1"
	locShare.TargetType = domain.ShareTargetLocation
	locShare.TargetID = "loc-1"

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var owner domain.User
	owner.ID = "owner-1"
	owner.Email = "owner@example.com"

	shareRepo := &mockShareRepoByType{locationShares: []domain.Share{locShare}}
	svc := newTestShareService(shareRepo, &mockUserStore{users: []domain.User{owner}}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc}, listByOwnerErr: domain.ErrIO})

	_, err := svc.ListSharedWithMe(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// mockShareRepoByType is a share repo that returns different shares per target type query.
type mockShareRepoByType struct {
	mockShareRepo
	itemShares     []domain.Share
	outfitShares   []domain.Share
	locationShares []domain.Share
	outfitListErr  error
	locationListErr error
}

func (m *mockShareRepoByType) ListByRecipientAndType(_ context.Context, _ string, targetType domain.ShareTargetType) ([]domain.Share, error) {
	switch targetType {
	case domain.ShareTargetItem:
		return m.itemShares, nil
	case domain.ShareTargetOutfit:
		if m.outfitListErr != nil {
			return nil, m.outfitListErr
		}
		return m.outfitShares, nil
	case domain.ShareTargetLocation:
		if m.locationListErr != nil {
			return nil, m.locationListErr
		}
		return m.locationShares, nil
	}
	return nil, nil
}

// ── Revoke ────────────────────────────────────────────────────────────────────

func TestShareServiceRevokeShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Revoke(ctx, "owner-1", "share-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareServiceRevokeShouldReturnErrNotFoundWhenShareDoesNotExist(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	err := svc.Revoke(t.Context(), "owner-1", "share-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareServiceRevokeShouldReturnErrForbiddenWhenCallerIsNotShareOwner(t *testing.T) {
	var share domain.Share
	share.ID = "share-1"
	share.OwnerID = "owner-2"

	shareRepo := &mockShareRepo{shares: []domain.Share{share}}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	err := svc.Revoke(t.Context(), "owner-1", "share-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestShareServiceRevokeShouldReturnErrorWhenRepoDeleteFails(t *testing.T) {
	var share domain.Share
	share.ID = "share-1"
	share.OwnerID = "owner-1"

	shareRepo := &mockShareRepo{shares: []domain.Share{share}, deleteErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	err := svc.Revoke(t.Context(), "owner-1", "share-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceRevokeShouldDeleteShareWhenCallerIsOwner(t *testing.T) {
	var share domain.Share
	share.ID = "share-1"
	share.OwnerID = "owner-1"

	shareRepo := &mockShareRepo{shares: []domain.Share{share}}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	err := svc.Revoke(t.Context(), "owner-1", "share-1")
	require.NoError(t, err)
	require.Empty(t, shareRepo.shares)
}

// ── HasReadAccess ─────────────────────────────────────────────────────────────

func TestShareServiceHasReadAccessShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.HasReadAccess(ctx, "user-1", domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareServiceHasReadAccessShouldReturnFalseWhenTargetTypeIsUnknown(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetType("unknown"), "entity-1")
	require.NoError(t, err)
	require.False(t, got)
}

func TestShareServiceHasReadAccessShouldReturnErrorWhenHasDirectAccessFails(t *testing.T) {
	shareRepo := &mockShareRepo{hasDirectAccessErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetOutfit, "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceHasReadAccessShouldReturnTrueWhenOutfitIsDirectlyShared(t *testing.T) {
	shareRepo := &mockShareRepo{hasDirectAccessResult: true}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetOutfit, "outfit-1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestShareServiceHasReadAccessShouldReturnFalseWhenOutfitIsNotShared(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetOutfit, "outfit-1")
	require.NoError(t, err)
	require.False(t, got)
}

func TestShareServiceHasReadAccessShouldReturnTrueWhenItemIsDirectlyShared(t *testing.T) {
	shareRepo := &mockShareRepo{hasDirectAccessResult: true}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetItem, "item-1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestShareServiceHasReadAccessShouldReturnFalseWhenItemIsNotSharedAtAll(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetItem, "item-1")
	require.NoError(t, err)
	require.False(t, got)
}

func TestShareServiceHasReadAccessShouldReturnErrorWhenItemGetFailsDuringLocationCheck(t *testing.T) {
	// no direct access; item.Get fails
	itemRepo := &mockItemRepo{getErr: domain.ErrIO}
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, itemRepo, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceHasReadAccessShouldReturnTrueWhenItemLocationIsShared(t *testing.T) {
	locID := "loc-1"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.LocationID = &locID

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	shareRepo := &mockShareRepo{}
	// HasDirectAccess returns true for location check
	shareRepo.hasDirectAccessResult = false
	// We need HasDirectAccess to return true when checking location.
	// Use a custom mock that returns true for location type.
	customShareRepo := &mockShareRepoSelectiveAccess{
		trueForType: domain.ShareTargetLocation,
		trueForID:   "loc-1",
	}
	svc := newTestShareService(customShareRepo, &mockUserStore{}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc}})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetItem, "item-1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestShareServiceHasReadAccessShouldReturnTrueWhenItemLocationAncestorIsShared(t *testing.T) {
	parentID := "loc-parent"
	locID := "loc-1"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.LocationID = &locID

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.ParentID = &parentID

	var parent domain.Location
	parent.ID = "loc-parent"
	parent.OwnerID = "owner-1"

	// Direct access only granted for ancestor
	customShareRepo := &mockShareRepoSelectiveAccess{
		trueForType: domain.ShareTargetLocation,
		trueForID:   "loc-parent",
	}
	svc := newTestShareService(customShareRepo, &mockUserStore{}, &mockItemRepo{items: []domain.Item{item}}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc, parent}})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetItem, "item-1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestShareServiceHasReadAccessShouldReturnTrueWhenLocationIsDirectlyShared(t *testing.T) {
	shareRepo := &mockShareRepo{hasDirectAccessResult: true}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetLocation, "loc-1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestShareServiceHasReadAccessShouldReturnTrueWhenLocationAncestorIsShared(t *testing.T) {
	parentID := "loc-parent"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.ParentID = &parentID

	var parent domain.Location
	parent.ID = "loc-parent"
	parent.OwnerID = "owner-1"

	customShareRepo := &mockShareRepoSelectiveAccess{
		trueForType: domain.ShareTargetLocation,
		trueForID:   "loc-parent",
	}
	svc := newTestShareService(customShareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc, parent}})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetLocation, "loc-1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestShareServiceHasReadAccessShouldReturnFalseWhenLocationIsNotShared(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{locations: []domain.Location{loc}})

	got, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetLocation, "loc-1")
	require.NoError(t, err)
	require.False(t, got)
}

func TestShareServiceHasReadAccessShouldReturnErrorWhenLocationGetFailsDuringAncestorWalk(t *testing.T) {
	parentID := "loc-parent"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.ParentID = &parentID

	locRepo := &mockLocationRepo{
		locations: []domain.Location{loc},
		getErrFor: map[string]error{"loc-parent": domain.ErrIO},
	}
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, locRepo)

	_, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetLocation, "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// mockShareRepoSelectiveAccess returns HasDirectAccess=true only for specific type+ID.
type mockShareRepoSelectiveAccess struct {
	mockShareRepo
	trueForType domain.ShareTargetType
	trueForID   string
}

func (m *mockShareRepoSelectiveAccess) HasDirectAccess(_ context.Context, _ string, targetType domain.ShareTargetType, targetID string) (bool, error) {
	return targetType == m.trueForType && targetID == m.trueForID, nil
}

func TestShareServiceHasReadAccessShouldReturnFalseWhenLocationTreeHasCycle(t *testing.T) {
	parent1 := "loc-2"
	parent2 := "loc-1"
	var loc1 domain.Location
	loc1.ID = "loc-1"
	loc1.ParentID = &parent1
	var loc2 domain.Location
	loc2.ID = "loc-2"
	loc2.ParentID = &parent2

	locRepo := &mockLocationRepo{locations: []domain.Location{loc1, loc2}}
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, locRepo)

	ok, err := svc.HasReadAccess(t.Context(), "user-1", domain.ShareTargetLocation, "loc-1")
	require.NoError(t, err)
	require.False(t, ok)
}

// ── collectDescendantIDs ──────────────────────────────────────────────────────

func TestCollectDescendantIDsShouldReturnOnlyRootWhenNoChildren(t *testing.T) {
	locs := []domain.Location{}
	got := collectDescendantIDs("root-1", locs)
	require.Equal(t, map[string]bool{"root-1": true}, got)
}

func TestCollectDescendantIDsShouldReturnRootAndDirectChildren(t *testing.T) {
	parent := "root-1"
	var child1 domain.Location
	child1.ID = "child-1"
	child1.ParentID = &parent
	var child2 domain.Location
	child2.ID = "child-2"
	child2.ParentID = &parent

	got := collectDescendantIDs("root-1", []domain.Location{child1, child2})
	require.Equal(t, map[string]bool{"root-1": true, "child-1": true, "child-2": true}, got)
}

func TestCollectDescendantIDsShouldReturnMultiLevelDescendants(t *testing.T) {
	grandparent := "root-1"
	parent := "child-1"
	var child domain.Location
	child.ID = "child-1"
	child.ParentID = &grandparent
	var grandchild domain.Location
	grandchild.ID = "grandchild-1"
	grandchild.ParentID = &parent

	got := collectDescendantIDs("root-1", []domain.Location{child, grandchild})
	require.Equal(t, map[string]bool{"root-1": true, "child-1": true, "grandchild-1": true}, got)
}

func TestCollectDescendantIDsShouldNotIncludeUnrelatedLocations(t *testing.T) {
	parent := "root-1"
	var child domain.Location
	child.ID = "child-1"
	child.ParentID = &parent
	otherParent := "other-root"
	var unrelated domain.Location
	unrelated.ID = "unrelated-1"
	unrelated.ParentID = &otherParent

	got := collectDescendantIDs("root-1", []domain.Location{child, unrelated})
	require.Equal(t, map[string]bool{"root-1": true, "child-1": true}, got)
}

// ── Create — unknown target type ──────────────────────────────────────────────

func TestShareServiceCreateShouldReturnErrValidationWhenTargetTypeIsUnknown(t *testing.T) {
	var recipient domain.User
	recipient.ID = "user-2"
	recipient.Email = "user2@example.com"

	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{users: []domain.User{recipient}}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	_, err := svc.Create(t.Context(), "owner-1", CreateShareInput{
		RecipientID: "user-2",
		TargetType:  domain.ShareTargetType("unknown"),
		TargetID:    "item-1",
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

// ── DeleteByTarget ────────────────────────────────────────────────────────────

func TestShareServiceDeleteByTargetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := newTestShareService(&mockShareRepo{}, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.DeleteByTarget(ctx, domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareServiceDeleteByTargetShouldReturnErrorWhenRepoFails(t *testing.T) {
	shareRepo := &mockShareRepo{deleteByTargetErr: domain.ErrIO}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	err := svc.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareServiceDeleteByTargetShouldDelegateToRepoWhenSuccessful(t *testing.T) {
	shareRepo := &mockShareRepo{}
	svc := newTestShareService(shareRepo, &mockUserStore{}, &mockItemRepo{}, &mockOutfitRepo{}, &mockLocationRepo{})

	err := svc.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item-1")
	require.NoError(t, err)
}

